package controllers

import (
	"github.com/astaxie/beego"
)

type Demo2Controller struct {
	beego.Controller
}

func (c *Demo2Controller) GetFunc() {
	c.Ctx.WriteString("GetFunc")
}
