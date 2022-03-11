package routers

import (
	"test_beego/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	// http://192.168.117.131:8080/zz1

	// 3.正则路由，这里用固定路由演示
	// （1）带或者不带？
	// 可以有，也可以没有id
	// http://localhost:8082/zz1
	// beego.Router("/zz1/?:id", &controllers.Demo1Controller{})
	// // http://localhost:8082/zz2/111
	// // 必须带id，不然报错
	// beego.Router("/zz2/:id", &controllers.Demo1Controller{})
	// // （2）数字匹配
	// // http://localhost:8082/zz3/333
	// beego.Router("/zz3/:id([0-9]+)", &controllers.Demo1Controller{})
	// // （3）字符串匹配
	// // http://localhost:8082/zz4/root
	// beego.Router("/zz4/:username([\\w]+)", &controllers.Demo1Controller{})

	// int类型匹配
	beego.Router("/zz8/my_:id([0-9]+)", &controllers.Demo1Controller{})

}
