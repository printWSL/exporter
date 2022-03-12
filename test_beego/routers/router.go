package routers

import (
	"test_beego/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/zx", &controllers.Demo2Controller{}, "GET:GetFunc")

}
