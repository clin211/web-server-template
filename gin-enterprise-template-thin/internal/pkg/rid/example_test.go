package rid_test

import (
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/pkg/rid"
)

func ExampleResourceID_String() {
	// 定义一个资源标识符，例如，用户资源。
	userID := rid.UserID

	// 调用 String 方法将 ResourceID 类型转换为 string 类型。
	idString := userID.String()

	// 输出结果。
	fmt.Println(idString)

	// Output:
	// user
}
