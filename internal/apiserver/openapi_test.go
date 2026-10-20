package apiserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// apiErrorDefinition 是统一错误响应包络对应的 definition 名，
// 由 protoc-gen-openapiv2 的 legacy 命名策略生成（apiserver.v1.APIError -> v1APIError）。
const apiErrorDefinition = "v1APIError"

// swaggerDocumentPath 指向 make protoc 直接产出的 Swagger 2.0 契约文件（零后处理）。
func swaggerDocumentPath() string {
	return filepath.Join("..", "..", "api", "openapi", "apiserver.swagger.json")
}

type swaggerSchema struct {
	Ref        string                    `json:"$ref"`
	Type       string                    `json:"type"`
	Properties map[string]*swaggerSchema `json:"properties"`
}

type swaggerResponse struct {
	Description string         `json:"description"`
	Schema      *swaggerSchema `json:"schema"`
}

type swaggerOperation struct {
	Security  []map[string][]string       `json:"security"`
	Responses map[string]*swaggerResponse `json:"responses"`
}

type swaggerDocument struct {
	Swagger             string                                  `json:"swagger"`
	Definitions         map[string]*swaggerSchema               `json:"definitions"`
	SecurityDefinitions map[string]json.RawMessage              `json:"securityDefinitions"`
	Paths               map[string]map[string]*swaggerOperation `json:"paths"`
}

func loadSwaggerDocument(t *testing.T) *swaggerDocument {
	t.Helper()
	payload, err := os.ReadFile(swaggerDocumentPath())
	if err != nil {
		t.Fatalf("read OpenAPI document: %v", err)
	}
	var document swaggerDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}
	return &document
}

func TestOpenAPIDocumentDeclaresSwaggerContract(t *testing.T) {
	document := loadSwaggerDocument(t)

	if document.Swagger != "2.0" {
		t.Fatalf("document version = %q; want Swagger 2.0（protoc-gen-openapiv2 直出产物，无转换层）", document.Swagger)
	}
	if _, ok := document.SecurityDefinitions["BearerAuth"]; !ok {
		t.Fatal("OpenAPI document does not define BearerAuth")
	}

	// 全量 $ref 可达性：openapiv2 对无法解析的注解 ref 仅打日志不报错，
	// 这里兜底确保契约文件中不存在悬空引用。
	payload, err := os.ReadFile(swaggerDocumentPath())
	if err != nil {
		t.Fatalf("read OpenAPI document: %v", err)
	}
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}
	var dangling []string
	var walk func(node any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			for key, child := range value {
				if key == "$ref" {
					if ref, ok := child.(string); ok {
						if _, defined := document.Definitions[filepath.Base(ref)]; !defined {
							dangling = append(dangling, ref)
						}
					}
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(raw)
	for _, ref := range dangling {
		t.Errorf("OpenAPI document contains dangling $ref %q", ref)
	}
}

func TestOpenAPIOperationsDeclareEnvelopeAndSecurity(t *testing.T) {
	document := loadSwaggerDocument(t)

	operations := []struct {
		path     string
		method   string
		envelope string // 期望 200 响应指向的包络 definition
		secured  bool   // 是否要求 BearerAuth
	}{
		{"/healthz", "get", "v1HealthzResponseEnvelope", false},
		{"/v1/auth/login", "post", "v1LoginResponseEnvelope", false},
		{"/v1/auth/refresh-token", "put", "v1LoginResponseEnvelope", true},
		{"/v1/auth/info", "get", "v1InfoResponseEnvelope", true},
		{"/v1/auth/logout", "post", "v1LogoutResponseEnvelope", true},
		{"/v1/users", "post", "v1CreateUserResponseEnvelope", true},
		{"/v1/users", "get", "v1ListUserResponseEnvelope", true},
		{"/v1/roles/{role_id}", "get", "v1RoleDetailEnvelope", true},
		{"/v1/users/{userID}/roles", "get", "v1UserRolesEnvelope", true},
		{"/v1/users/{userID}", "get", "v1GetUserResponseEnvelope", true},
		{"/v1/users/{userID}", "put", "v1UpdateUserResponseEnvelope", true},
		{"/v1/users/{userID}", "delete", "v1DeleteUserResponseEnvelope", true},
		{"/v1/users/{userID}/change-password", "put", "v1ChangePasswordResponseEnvelope", true},
	}

	for _, tc := range operations {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			operation := document.Paths[tc.path][tc.method]
			if operation == nil {
				t.Fatalf("OpenAPI document does not describe %s %s", tc.method, tc.path)
			}

			response := operation.Responses["200"]
			if response == nil {
				t.Fatalf("%s %s does not describe a 200 response", tc.method, tc.path)
			}
			if want := "#/definitions/" + tc.envelope; response.Schema == nil || response.Schema.Ref != want {
				t.Errorf("%s %s 200 response ref = %v; want %s", tc.method, tc.path, response.Schema, want)
			}

			definition := operation.Responses["default"]
			if definition == nil {
				t.Fatalf("%s %s does not describe a default error response", tc.method, tc.path)
			}
			if want := "#/definitions/" + apiErrorDefinition; definition.Schema == nil || definition.Schema.Ref != want {
				t.Errorf("%s %s default response ref = %v; want %s", tc.method, tc.path, definition.Schema, want)
			}

			if secured := len(operation.Security) > 0; secured != tc.secured {
				t.Errorf("%s %s security = %v; want secured=%t", tc.method, tc.path, operation.Security, tc.secured)
			}
		})
	}
}

func TestOpenAPIEnvelopeSchemasMatchRuntimeResponse(t *testing.T) {
	document := loadSwaggerDocument(t)

	envelopes := []struct {
		envelope string // 包络 definition 名
		data     string // data 字段应指向的业务 schema
	}{
		{"v1HealthzResponseEnvelope", "v1HealthzResponse"},
		{"v1LoginResponseEnvelope", "v1LoginResponse"},
		{"v1InfoResponseEnvelope", "v1InfoResponse"},
		{"v1RoleDetailEnvelope", "v1RoleDetail"},
		{"v1UserRolesEnvelope", "v1UserRoles"},
		{"v1LogoutResponseEnvelope", "v1LogoutResponse"},
		{"v1CreateUserResponseEnvelope", "v1CreateUserResponse"},
		{"v1GetUserResponseEnvelope", "v1GetUserResponse"},
		{"v1UpdateUserResponseEnvelope", "v1UpdateUserResponse"},
		{"v1DeleteUserResponseEnvelope", "v1DeleteUserResponse"},
		{"v1ListUserResponseEnvelope", "v1ListUserResponse"},
		{"v1ChangePasswordResponseEnvelope", "v1ChangePasswordResponse"},
	}

	for _, tc := range envelopes {
		t.Run(tc.envelope, func(t *testing.T) {
			schema, ok := document.Definitions[tc.envelope]
			if !ok {
				t.Fatalf("OpenAPI document does not define %s", tc.envelope)
			}
			if len(schema.Properties) != 3 {
				t.Fatalf("%s properties = %v; want exactly code/message/data", tc.envelope, schema.Properties)
			}
			for _, field := range []string{"code", "message", "data"} {
				if _, ok := schema.Properties[field]; !ok {
					t.Errorf("%s is missing property %q", tc.envelope, field)
				}
			}
			if got := schema.Properties["data"].Ref; got != "#/definitions/"+tc.data {
				t.Errorf("%s data ref = %q; want %q", tc.envelope, got, tc.data)
			}
		})
	}

	// 错误包络：仅 code/message/reason，与 errorsx.FromBizError 的输出一致（无 data）。
	errorSchema, ok := document.Definitions[apiErrorDefinition]
	if !ok {
		t.Fatalf("OpenAPI document does not define %s", apiErrorDefinition)
	}
	if len(errorSchema.Properties) != 3 {
		t.Errorf("%s properties = %v; want exactly code/message/reason", apiErrorDefinition, errorSchema.Properties)
	}

	// 敏感信息回归：登录响应不暴露当前用户资料，info 响应包含当前用户。
	if loginSchema, ok := document.Definitions["v1LoginResponse"]; ok {
		if _, exposes := loginSchema.Properties["user"]; exposes {
			t.Error("OpenAPI login response unexpectedly exposes current user")
		}
	} else {
		t.Fatal("OpenAPI document does not define v1LoginResponse")
	}
	if infoSchema, ok := document.Definitions["v1InfoResponse"]; ok {
		if _, describes := infoSchema.Properties["user"]; !describes {
			t.Error("OpenAPI info response does not describe current user")
		}
	} else {
		t.Fatal("OpenAPI document does not define v1InfoResponse")
	}
}
