package controllers

import (
	"github.com/astaxie/beego"
)

type Demo1Controller struct {
	beego.Controller
}

func (c *Demo1Controller) Get() {
	id := c.Ctx.Input.Param(":id")
	c.Ctx.WriteString("id=" + id + "\r\n")

}
