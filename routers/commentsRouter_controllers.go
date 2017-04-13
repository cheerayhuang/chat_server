package routers

import (
	"github.com/astaxie/beego"
)

func init() {

	beego.GlobalControllerRouter["chat_server/controllers:ChatController"] = append(beego.GlobalControllerRouter["chat_server/controllers:ChatController"],
		beego.ControllerComments{
			Method: "WSConnect",
			Router: `/`,
			AllowHTTPMethods: []string{"get"},
			Params: nil})

}
