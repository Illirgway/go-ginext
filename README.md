
# Go [Gin](https://github.com/gin-gonic/gin) Extender package

Extends functionality for [Gin - fast web framework](https://github.com/gin-gonic/gin) 

## Usage

### Automatic registration of complex controllers

Instead of registering many single handler function, use controller type and its methods to automatically
create appropriate gin router paths using the below functionality

#### Controller type, its instance (object) and endpoint methods

`Controller` is an any type (and so its instance), which have at least one `endpoint` method.
`endpoint` is a common type method (with `Controller` type receiver) of `gin.HandlerFunc` signature
`func (*gin.Context)` and specific format of its name: `EpMethodName = <HttpMethod><Endpoint>`, where
`<HttpMethod>` is one of RFC7231 http methods `[Get Head Post Put Delete Connect Options Trace]` and
`<Endpoint>` is standard go method name CamelCase suffix, which will be final endpoint of full abs 
router path to this method, automagically converting from `CamelCase` to `kebab-case`, e.g. 

```gotemplate

type ControllerExample struct {}

func (c *ControllerExample) GetUserName(ctx *gin.Context) {
	ctx.String(http.StatusOK, "hello from GetUserName")
}

func main () {

	// ...

	r := gin.New()
	
	c := &ControllerExample{}
	
	ginext.EmbedController(r, c)
	
	// now after starting r listening on port 8080, 
	// you can request GET `http://localhost:8080/user-name` ==> "hello from GetUserName"
	
	// ...
}
```

If you want to attach `Controller` to source `IRoute` as base segment of `endpoint`s set, use `ginext.AttachController` -
kebab-case'd controller name without optional `Controller` prefix would be used as segment name:

```gotemplate

type ControllerCommonSegment struct {}

func (c *ControllerCommonSegment) PostSaveData(ctx *gin.Context) {
	ctx.String(http.StatusOK, "hello from PostSaveData")
}

// can use empty name to register "" (index) endpoint
func (c *ControllerCommonSegment) Get(ctx *gin.Context) {
	ctx.String(http.StatusOK, "hello from Get - root [index] endpoint")
}


func main () {

	// ...

	r := gin.New()
	
	c := &ControllerCommonSegment{}
	
	ginext.AttachController(r, c)
	
	// now after starting r listening on port 8080, 
	// GET `http://localhost:8080/common-segment/` ==> "hello from Get - root [index] endpoint"
	// POST `http://localhost:8080/common-segment/save-data` ==> "hello from PostSaveData"
	
	// ...
}
```

#### Additional optional wrapper handlers

You may use special Controller's methods `Init`, `Before` and `After` for additional control:

```gotemplate
type ControllerWithWrappers struct {}

func (c *ControllerWithWrappers) Init() error {
	fmt.Println("this is init")
	return nil
}

func (c *ControllerWithWrappers) Before(ctx *gin.Context) {
	fmt.Println("before")
}

func (c *ControllerWithWrappers) Get(ctx *gin.Context) {
	fmt.Println("get")
}

func (c *ControllerWithWrappers) After(ctx *gin.Context) {
	fmt.Println("after")
}

func main () {

	// ...

	r := gin.New()
	
	c := &ControllerWithWrappers{}
	
	ginext.AttachController(r, c) // immediately prints "this is init"
	
	// now after starting r listening on port 8080, 
	// you can request GET `http://localhost:8080/user-name/` ==> prints "before" "get" "after", each on new line
	
	// ...
}
```

`Init` is special method of signature `ControllerInitMethod = func() error`, which is calling when controller starting register in router and
may returns error, which immediately breaks registration and returns to a superior caller

`Before` and `After` is the common Gin HandlerFunc, which wrapped every registered explicit handler of the controller

#### Append trailing slashes to controller method `endpoint`s on registration

Use `ginext.AppendTrailingSlash(true)` before registration by `AttachController` / `EmbedController` to enable 
automatic trailing slash append

```gotemplate
type ControllerExample struct {}

func (c *ControllerExample) GetUserName(ctx *gin.Context) {
	ctx.String(http.StatusOK, "hello from GetUserName")
}

func main () {

	// ...

	r := gin.New()
	
	c := &ControllerExample{}
	
	ginext.AppendTrailingSlash(true) // before `ginext.EmbedController` 
	
	ginext.EmbedController(r, c)
	
	// now after starting r listening on port 8080, 
	// you can request GET `http://localhost:8080/user-name/` ==> "hello from GetUserName"
	
	// ...
}
```

#### Any Gin router type

`AppendController` and `EmbedController` use gin interface `gin.IRoutes` as `router` arg, so you may use either `gin.Engine` or 
`gin.RouterGroup`, including groups with attached middleware (e.g. `auth` and `session` mws) and path's prefix

#### Additional examples

See `router_test.go`

### TODO

* more tests required
* implement endpoint ctx Params (the easiest way is to append `CatchAll` option to `endpoint`s paths)

## LICENSE

This program is free software: you can redistribute it and/or modify it under the terms of the 
GNU General Public License as published by the Free Software Foundation, either version 3 of the License, 
or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied 
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.
If not, see <https://www.gnu.org/licenses/>.

Copyright &copy; 2023 Illirgway