package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

    beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Login",
            Router: `/login`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Logout",
            Router: `/logout`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Refresh",
            Router: `/logout`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"],
        beego.ControllerComments{
            Method: "GetMaintenance",
            Router: `/maintenance`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Maintenance",
            Router: `/maintenance`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "Post",
            Router: `/:table`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "Put",
            Router: `/:table`,
            AllowHTTPMethods: []string{"put"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "Delete",
            Router: `/:table`,
            AllowHTTPMethods: []string{"delete"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "Get",
            Router: `/:table`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "Math",
            Router: `/:table/:function`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:GenericController"],
        beego.ControllerComments{
            Method: "WebSocket",
            Router: `/:table/websocket`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:MainController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:MainController"],
        beego.ControllerComments{
            Method: "Main",
            Router: `/`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["sqldb-ws/controllers:MainController"] = append(beego.GlobalControllerRouter["sqldb-ws/controllers:MainController"],
        beego.ControllerComments{
            Method: "Download",
            Router: `/download/:path`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
