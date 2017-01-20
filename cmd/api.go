// Copyright © 2017 edwin <edwin.lzh@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"time"

	"github.com/iris-contrib/graceful"
	"github.com/iris-contrib/middleware/loggerzap"
	"github.com/iris-contrib/middleware/recovery"
	"github.com/kataras/iris"
	"github.com/lvzhihao/wxaapp/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "开放接口",
	Long: `GET /api/openid 
GET /api/sessionid
GET /api/session
PUT /api/session
GET /api/userinfo
PUT /api/userinfo`,
	Run: func(cmd *cobra.Command, args []string) {
		app := iris.New()

		//global recovery
		app.Use(loggerzap.New(loggerzap.Config{
			Status: true,
			IP:     true,
			Method: true,
			Path:   true,
		}))
		app.Use(recovery.Handler)

		//method sign
		//app.Post("/router?method={method}", api.Router)

		//rest，no sign,
		app.Get("/api/openid", api.GetOpenId)
		app.Get("/api/sessionid", api.GetSessionId)
		app.Get("/api/session", api.GetSession)
		app.Put("/api/session", api.PutSession)
		app.Get("/api/userinfo", api.GetUserInfo)
		app.Put("/api/userinfo", api.PutUserInfo)

		graceful.Run(viper.GetString("api_host")+":"+viper.GetString("api_port"), time.Duration(10)*time.Second, app)
	},
}

func init() {
	RootCmd.AddCommand(apiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// apiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// apiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
