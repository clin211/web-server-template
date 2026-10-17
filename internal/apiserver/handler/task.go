package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"time"
)

func installTaskRoutes(v1 *gin.RouterGroup, h *Handler) {
	g := v1.Group("", h.mws...)
	s := h.biz.TasksV1()
	installReportRoutes(g, s)
	installScheduleRoutes(g, s)
	installRBACRoutes(g, s)
	installDeadLetterRoutes(g, h, s)
}
func installReportRoutes(g *gin.RouterGroup, s *taskbiz.Service) {
	g.POST("/reports", func(c *gin.Context) {
		var p job.ReportParams
		if !strictBody(c, &p) {
			return
		}
		t, err := s.Submit(c.Request.Context(), p, c.GetHeader("Idempotency-Key"))
		reply(c, 202, t, err)
	})
	g.GET("/tasks", listTasks(s))
	g.GET("/tasks/:taskID", func(c *gin.Context) {
		t, err := s.Get(c.Request.Context(), c.Param("taskID"), "task.view")
		reply(c, 200, t, err)
	})
	g.GET("/tasks/:taskID/attempts", func(c *gin.Context) {
		a, err := s.Attempts(c.Request.Context(), c.Param("taskID"))
		reply(c, 200, gin.H{"items": a}, err)
	})
	g.POST("/tasks/:taskID/cancel", func(c *gin.Context) { reply(c, 200, nil, s.Cancel(c.Request.Context(), c.Param("taskID"))) })
	g.POST("/tasks/:taskID/retry", func(c *gin.Context) {
		t, err := s.Retry(c.Request.Context(), c.Param("taskID"), c.GetHeader("Idempotency-Key"))
		reply(c, 202, t, err)
	})
	g.GET("/tasks/:taskID/result", func(c *gin.Context) {
		r, err := s.Download(c.Request.Context(), c.Param("taskID"))
		if err != nil {
			reply(c, 0, nil, err)
			return
		}
		c.Header("Content-Disposition", "attachment; filename=report.csv")
		c.Header("Cache-Control", "no-store")
		c.Data(200, r.ContentType, r.Content)
	})
	g.GET("/report-notifications", func(c *gin.Context) {
		items, err := s.Notifications(c.Request.Context())
		reply(c, 200, gin.H{"items": items}, err)
	})
}
func installScheduleRoutes(g *gin.RouterGroup, s *taskbiz.Service) {
	g.POST("/schedules", func(c *gin.Context) {
		var p taskbiz.ScheduleInput
		if !strictBody(c, &p) {
			return
		}
		m, err := s.SaveSchedule(c.Request.Context(), "", p)
		reply(c, 201, m, err)
	})
	g.GET("/schedules", func(c *gin.Context) {
		items, err := s.Schedules(c.Request.Context(), c.Query("all") == "true")
		reply(c, 200, gin.H{"items": items}, err)
	})
	g.GET("/schedules/:scheduleID", func(c *gin.Context) {
		m, err := s.Schedule(c.Request.Context(), c.Param("scheduleID"))
		reply(c, 200, m, err)
	})
	g.PUT("/schedules/:scheduleID", func(c *gin.Context) {
		var p taskbiz.ScheduleInput
		if !strictBody(c, &p) {
			return
		}
		m, err := s.SaveSchedule(c.Request.Context(), c.Param("scheduleID"), p)
		reply(c, 200, m, err)
	})
	g.GET("/schedules/:scheduleID/history", func(c *gin.Context) {
		items, err := s.ScheduleHistory(c.Request.Context(), c.Param("scheduleID"))
		reply(c, 200, gin.H{"items": items}, err)
	})
	for _, action := range []string{"pause", "resume", "archive"} {
		g.POST("/schedules/:scheduleID/"+action, func(c *gin.Context) {
			var p struct {
				Version int64 `json:"version"`
			}
			if !strictBody(c, &p) {
				return
			}
			reply(c, 200, nil, s.ScheduleAction(c.Request.Context(), c.Param("scheduleID"), action, p.Version))
		})
	}
}
func installRBACRoutes(g *gin.RouterGroup, s *taskbiz.Service) {
	for _, kind := range []string{"roles", "permissions", "audits", "dead-letters"} {
		g.GET("/"+kind, func(c *gin.Context) {
			items, err := s.AdminList(c.Request.Context(), kind)
			reply(c, 200, gin.H{"items": items}, err)
		})
	}
	g.POST("/roles", func(c *gin.Context) {
		var p struct {
			RoleID string `json:"role_id"`
			Name   string `json:"name"`
		}
		if !strictBody(c, &p) {
			return
		}
		reply(c, 201, nil, s.RoleMutation(c.Request.Context(), "create", p.RoleID, p.Name, nil))
	})
	g.PUT("/roles/:roleID/permissions", func(c *gin.Context) {
		var p struct {
			Permissions []string `json:"permissions"`
		}
		if !strictBody(c, &p) {
			return
		}
		reply(c, 200, nil, s.RoleMutation(c.Request.Context(), "permissions", c.Param("roleID"), "", p.Permissions))
	})
	for _, action := range []string{"assign", "revoke"} {
		g.POST("/roles/:roleID/"+action, func(c *gin.Context) {
			var p struct {
				UserID string `json:"user_id"`
			}
			if !strictBody(c, &p) {
				return
			}
			reply(c, 200, nil, s.RoleMutation(c.Request.Context(), action, c.Param("roleID"), p.UserID, nil))
		})
	}
}
func installDeadLetterRoutes(g *gin.RouterGroup, h *Handler, s *taskbiz.Service) {
	g.GET("/dead-letters/:deadLetterID", func(c *gin.Context) {
		item, err := s.DeadLetter(c.Request.Context(), c.Param("deadLetterID"))
		reply(c, 200, item, err)
	})
	g.POST("/dead-letters/:deadLetterID/replay", func(c *gin.Context) {
		if !h.allowReplay(c) {
			return
		}
		id, err := s.Replay(c.Request.Context(), c.Param("deadLetterID"), c.GetHeader("Idempotency-Key"))
		reply(c, 202, gin.H{"event_id": id}, err)
	})
	g.POST("/dead-letters/replay", h.batchReplay(s))
}

func strictBody(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		reply(c, 0, nil, errno.ErrInvalidArgument)
		return false
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		reply(c, 0, nil, errno.ErrInvalidArgument)
		return false
	}
	return true
}
func reply(c *gin.Context, status int, data any, err error) {
	if errors.Is(err, store.ErrIdempotencyConflict) {
		c.JSON(409, gin.H{"code": 409, "message": "resource version, state or idempotency conflict", "reason": "Task.Conflict"})
		return
	}
	if err != nil {
		core.WriteResponse(c, nil, err)
		return
	}
	if data == nil {
		data = gin.H{}
	}
	c.JSON(status, gin.H{"code": 0, "message": "success", "data": data})
}

func (h *Handler) allowReplay(c *gin.Context) bool {
	if err := h.biz.TasksV1().Require(c.Request.Context(), "dlq.replay", ""); err != nil {
		reply(c, 0, nil, err)
		return false
	}
	if h.replayLimiter == nil {
		reply(c, 0, nil, errno.ErrServiceUnavailable)
		return false
	}
	ok, err := h.replayLimiter.AllowLogin(c.Request.Context(), "dlq:"+contextx.UserID(c.Request.Context()), 10, time.Minute)
	if err != nil {
		reply(c, 0, nil, errno.ErrServiceUnavailable)
		return false
	}
	if !ok {
		if err := h.biz.TasksV1().RejectManagement(c.Request.Context(), "dlq.rate_limited", ""); err != nil {
			reply(c, 0, nil, errno.ErrServiceUnavailable)
			return false
		}
		c.Header("Retry-After", "60")
		reply(c, 0, nil, errno.ErrTooManyRequests)
		return false
	}
	return true
}

func listTasks(s *taskbiz.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		before, err := taskPage(c.Query("page_token"))
		if err != nil {
			reply(c, 0, nil, err)
			return
		}
		items, err := s.List(c.Request.Context(), c.Query("all") == "true", before)
		next := ""
		if len(items) == 100 {
			next = base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(items[len(items)-1].ID, 10)))
		}
		reply(c, 200, gin.H{"items": items, "page_token": next}, err)
	}
}

func (h *Handler) batchReplay(s *taskbiz.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.allowReplay(c) {
			return
		}
		var p struct {
			IDs []string `json:"ids"`
		}
		if !strictBody(c, &p) {
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if len(p.IDs) < 1 || len(p.IDs) > 20 || key == "" || len(key) > 80 {
			reply(c, 0, nil, errno.ErrInvalidArgument)
			return
		}
		out := replayItems(c, s, p.IDs, key)
		reply(c, 202, gin.H{"items": out}, nil)
	}
}

func taskPage(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, errno.ErrInvalidArgument
	}
	before, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil || before < 1 {
		return 0, errno.ErrInvalidArgument
	}
	return before, nil
}

func replayItems(c *gin.Context, s *taskbiz.Service, ids []string, key string) []gin.H {
	out := make([]gin.H, 0, len(ids))
	for _, id := range ids {
		event, err := s.Replay(c.Request.Context(), id, key+":"+id)
		item := gin.H{"dead_letter_id": id, "event_id": event, "success": err == nil}
		if err != nil {
			item["reason"] = "replay_rejected"
		}
		out = append(out, item)
	}
	return out
}
