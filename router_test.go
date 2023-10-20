/**
 * This file is part of the go ginext package (https://github.com/Illirgway/go-ginext)
 *
 * Copyright (c) 2023 Illirgway
 *
 * This program is free software: you can redistribute it and/or modify it under the terms of the GNU
 * General Public License as published by the Free Software Foundation, either version 3 of the License,
 * or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
 * without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 * See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with this program.
 * If not, see <https://www.gnu.org/licenses/>.
 *
 */

package ginext

import (
	"errors"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type testCase struct {
	method   string
	path     string
	status   int
	response string
}

// ========
// root controller
type ControllerRoot struct {
	t *testing.T
}

//

func (c *ControllerRoot) helper(ctx *gin.Context, m string) {

	c.t.Logf("%p %d %s", c, http.StatusOK, m)

	if ctx != nil {
		ctx.String(http.StatusOK, m)
	}
}

func (c *ControllerRoot) Post(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.POST [index]")
}

func (c *ControllerRoot) Get(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.GET [index]")
}

func (c *ControllerRoot) GetEndpoint(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.GET:Endpoint")
}

func (c *ControllerRoot) PostEndpoint(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.POST:Endpoint")
}

func (c *ControllerRoot) PostOnlyMethod(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.POST:OnlyMethod")
}

func (c *ControllerRoot) ActionKnown(ctx *gin.Context) {
	c.helper(ctx, "ControllerRoot.Action:Known")
}

func (c *ControllerRoot) IgnoreThisMethod() {
	c.t.Errorf("ControllerRoot.IgnoreThisMethod executed!")
}

func (c *ControllerRoot) IgnoreThisMethod2(ctx *gin.Context) {
	c.t.Errorf("ControllerRoot.IgnoreThisMethod2 executed!")
}

// =======
// controller with wrong method
type ControllerBadMethod struct {
	t *testing.T
}

func (c *ControllerBadMethod) GetGoodMethod(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
}

func (c *ControllerBadMethod) PostBadMethod(ctx *gin.Context, err error) {
	ctx.Status(http.StatusOK)
}

// ========
// common controller
type ControllerCommon struct {
	t     *testing.T
	quiet bool
}

func (c *ControllerCommon) helper(ctx *gin.Context, m string) {

	if !c.quiet {
		c.t.Logf("%p %d %s", c, http.StatusOK, m)
	}

	if ctx != nil {
		ctx.String(http.StatusOK, m)
	}
}

func (c *ControllerCommon) Get(ctx *gin.Context) {
	c.helper(ctx, "ControllerCommon.GET [index]")
}

func (c *ControllerCommon) GetEndpoint(ctx *gin.Context) {
	c.helper(ctx, "ControllerCommon.GET:Endpoint")
}

func (c *ControllerCommon) PostEndpoint(ctx *gin.Context) {
	c.helper(ctx, "ControllerCommon.POST:Endpoint")
}

func (c *ControllerCommon) PostData(ctx *gin.Context) {
	c.helper(ctx, "ControllerCommon.POST:Data")
}

func (c *ControllerCommon) ActionKnown(ctx *gin.Context) {
	c.helper(ctx, "ControllerCommon.Action:Known")
}

// ==============

type ControllerEmbedded struct {
	t     *testing.T
	quiet bool
}

func (c *ControllerEmbedded) helper(ctx *gin.Context, m string) {

	if !c.quiet {
		c.t.Logf("%p %d %s", c, http.StatusOK, m)
	}

	if ctx != nil {
		ctx.String(http.StatusOK, m)
	}
}

func (c *ControllerEmbedded) ActionInternal(ctx *gin.Context) {
	c.helper(ctx, "ControllerEmbedded.Action:Internal")
}

func (c *ControllerEmbedded) ActionOverride(ctx *gin.Context) {
	c.helper(ctx, "ControllerEmbedded.Action:Override [original]")
}

// ----

type ControllerCompound struct {
	ControllerEmbedded
}

func (c *ControllerCompound) ActionExternal(ctx *gin.Context) {
	c.helper(ctx, "ControllerCompound.Action:External")
}

func (c *ControllerCompound) ActionOverride(ctx *gin.Context) {
	c.helper(ctx, "ControllerCompound.Action:Override [override]")
}

// ==============

func helperCheckTestCaseResponseResult(t *testing.T, tcase *testCase, rr *httptest.ResponseRecorder) bool {

	r := rr.Result()
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		t.Errorf("Request \"%s %s\" response body read error: %v", tcase.method, tcase.path, err)
		return false
	}

	if (r.StatusCode != tcase.status) || (string(body) != tcase.response) {
		t.Errorf("Request \"%s %s\" response mismatch error: want < %d => %q >, got < %d => %q >",
			tcase.method, tcase.path,
			tcase.status, tcase.response,
			r.StatusCode, string(body))
		return false
	}

	return true
}

const (
	errStr404 = "404 page not found"
	errStr405 = "405 method not allowed"
)

func helperRunTestsForRouter(t *testing.T, r *gin.Engine, cases []*testCase) {

	for _, tcase := range cases {

		t.Logf("\n%s %s\n", tcase.method, tcase.path)

		request := httptest.NewRequest(tcase.method, tcase.path, nil)

		// t.Logf("%#v", request)
		t.Logf("Request: %s %s", request.Method, request.URL.String())

		// https://blog.questionable.services/article/testing-http-handlers-go/
		writer := httptest.NewRecorder()

		r.ServeHTTP(writer, request)

		/* if status := writer.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusOK)
		} */

		t.Logf("== [%d] ==> %q", writer.Code, writer.Body.String())

		if !helperCheckTestCaseResponseResult(t, tcase, writer) {
			return
		}
	}
}

func newRouter() (r *gin.Engine) {

	r = gin.New()

	r.HandleMethodNotAllowed = true

	return r
}

func testControllerInternal1(t *testing.T, c interface{}, prepend bool, tests []*testCase) {

	r := newRouter()

	// indirect error
	if r == nil {
		t.Error("newRouter() returns nil")
		return
	}

	err := registerController(r, c, prepend)

	if err != nil {
		t.Error(err)
		return
	}

	helperRunTestsForRouter(t, r, tests)
}

//
// go test -count=1 -v -run "^TestRegisterControllerCommon1$"

var testRouterControllerCommonValues = []*testCase{
	{http.MethodGet, "/", http.StatusNotFound, errStr404},
	//
	{http.MethodGet, "/common/", http.StatusOK, "ControllerCommon.GET [index]"},
	{http.MethodGet, "/common/endpoint", http.StatusOK, "ControllerCommon.GET:Endpoint"},
	{http.MethodPost, "/common/endpoint", http.StatusOK, "ControllerCommon.POST:Endpoint"},
	{http.MethodPut, "/common/endpoint", http.StatusMethodNotAllowed, errStr405},
	//
	{http.MethodGet, "/common/data", http.StatusMethodNotAllowed, errStr405},
	{http.MethodPost, "/common/data", http.StatusOK, "ControllerCommon.POST:Data"},
	//
	{http.MethodGet, "/common/known", http.StatusOK, "ControllerCommon.Action:Known"},
	{http.MethodPost, "/common/known", http.StatusOK, "ControllerCommon.Action:Known"},
	{http.MethodPut, "/common/known", http.StatusOK, "ControllerCommon.Action:Known"},
	//
	{http.MethodGet, "/common/ignore-this-method", http.StatusNotFound, errStr404},
	{http.MethodPost, "/common/ignore-this-method2", http.StatusNotFound, errStr404},
	//
	{http.MethodGet, "/common/unknown", http.StatusNotFound, errStr404},
	{http.MethodPost, "/common/unknown", http.StatusNotFound, errStr404},
}

func TestRegisterControllerCommon1(t *testing.T) {
	testControllerInternal1(t, &ControllerCommon{t: t}, true, testRouterControllerCommonValues)
}

//
// go test -count=1 -v -run TestTreeControllerRoot1

var (
	testControllerRoot1Values = []*testCase{
		{http.MethodGet, "/", http.StatusOK, "ControllerRoot.GET [index]"},
		{http.MethodPost, "/", http.StatusOK, "ControllerRoot.POST [index]"},
		//
		{http.MethodGet, "/endpoint", http.StatusOK, "ControllerRoot.GET:Endpoint"},
		{http.MethodPost, "/endpoint", http.StatusOK, "ControllerRoot.POST:Endpoint"},
		{http.MethodPut, "/endpoint", http.StatusMethodNotAllowed, errStr405},
		//
		{http.MethodGet, "/onlymethod", http.StatusNotFound, errStr404},
		{http.MethodGet, "/only-method", http.StatusMethodNotAllowed, errStr405},
		{http.MethodPost, "/only-method", http.StatusOK, "ControllerRoot.POST:OnlyMethod"},
		//
		{http.MethodGet, "/known", http.StatusOK, "ControllerRoot.Action:Known"},
		{http.MethodGet, "/known/", http.StatusMovedPermanently, "<a href=\"/known\">Moved Permanently</a>.\n\n"},
		{http.MethodPost, "/known", http.StatusOK, "ControllerRoot.Action:Known"},
		{http.MethodPost, "/known/", http.StatusTemporaryRedirect, ""},
		{http.MethodPut, "/known", http.StatusOK, "ControllerRoot.Action:Known"},
		//
		{http.MethodPost, "/unknown", http.StatusNotFound, errStr404},
		//
		{http.MethodGet, "/long/endpoint/", http.StatusNotFound, errStr404},
	}
)

func TestRegisterControllerRoot1(t *testing.T) {
	testControllerInternal1(t, &ControllerRoot{t}, false, testControllerRoot1Values)
}

//
// go test -count=1 -v -run TestRegisterControllerCompound1

var (
	testControllerCompound1Values = []*testCase{
		{http.MethodGet, "/", http.StatusNotFound, errStr404},
		{http.MethodPost, "/", http.StatusNotFound, errStr404},
		//
		{http.MethodGet, "/compound/internal", http.StatusOK, "ControllerEmbedded.Action:Internal"},
		{http.MethodPost, "/compound/internal", http.StatusOK, "ControllerEmbedded.Action:Internal"},
		{http.MethodPost, "/compound/internal/", http.StatusTemporaryRedirect, ""},
		{http.MethodPut, "/internal", http.StatusNotFound, errStr404},
		//
		{http.MethodGet, "/compound/external", http.StatusOK, "ControllerCompound.Action:External"},
		{http.MethodPost, "/compound/external", http.StatusOK, "ControllerCompound.Action:External"},
		{http.MethodGet, "/compound/external/", http.StatusMovedPermanently, "<a href=\"/compound/external\">Moved Permanently</a>.\n\n"},
		{http.MethodPost, "/external", http.StatusNotFound, errStr404},
		//
		{http.MethodGet, "/compound/override", http.StatusOK, "ControllerCompound.Action:Override [override]"},
		{http.MethodPost, "/compound/override", http.StatusOK, "ControllerCompound.Action:Override [override]"},
		{http.MethodPut, "/compound/override/", http.StatusTemporaryRedirect, ""},
		{http.MethodPost, "/override", http.StatusNotFound, errStr404},
		//
		{http.MethodPost, "/unknown", http.StatusNotFound, errStr404},
		//
		{http.MethodGet, "/long/endpoint/", http.StatusNotFound, errStr404},
	}
)

func TestRegisterControllerCompound1(t *testing.T) {

	c := &ControllerCompound{ControllerEmbedded{t, false}}

	testControllerInternal1(t, c, true, testControllerCompound1Values)
}

//
// ==============

type ControllerTestInit struct {
	t *testing.T
}

func (c *ControllerTestInit) Init() error {
	return errors.New("ControllerTestInit.Init works fine!")
}

func (c *ControllerTestInit) Get(ctx *gin.Context) {
	ctx.String(http.StatusBadRequest, "ControllerTestInit.Get Should not happen")
}

type ControllerTestBefore struct {
	t *testing.T
}

func (c *ControllerTestBefore) Before(ctx *gin.Context) {
	ctx.String(http.StatusTeapot, "ControllerTestBefore.Before")
	ctx.Abort()
}

func (c *ControllerTestBefore) Get(ctx *gin.Context) {
	ctx.String(http.StatusBadRequest, "ControllerTestBefore.Get Should not happen")
}

type ControllerTestAfter struct {
	t *testing.T
}

func (c *ControllerTestAfter) After(ctx *gin.Context) {
	c.t.Log("ControllerTestAfter.After")
	ctx.String(http.StatusTeapot, "ControllerTestAfter.After")
}

func (c *ControllerTestAfter) Get(ctx *gin.Context) {
	//no-op
}

type ControllerTestWrappers struct {
	t *testing.T
	a []string
}

func (c *ControllerTestWrappers) Init() error {
	c.a = append(c.a, "init")
	return nil
}

func (c *ControllerTestWrappers) Before(ctx *gin.Context) {
	c.a = append(c.a, "before")
}

func (c *ControllerTestWrappers) After(ctx *gin.Context) {
	c.a = append(c.a, "after")
}

func (c *ControllerTestWrappers) PutIndex(ctx *gin.Context) {
	c.a = append(c.a, "handler")
	ctx.String(http.StatusAccepted, "ControllerTestWrappers.GetIndex")
}

var (
	testControllerWithWrappers1Before = []*testCase{
		{http.MethodGet, "/test-before/", http.StatusTeapot, "ControllerTestBefore.Before"},
	}

	testControllerWithWrappers1After = []*testCase{
		{http.MethodGet, "/test-after/", http.StatusTeapot, "ControllerTestAfter.After"},
	}

	testControllerWithWrappers1Wrappers = []*testCase{
		{http.MethodPut, "/test-wrappers/index", http.StatusAccepted, "ControllerTestWrappers.GetIndex"},
	}
)

// go test -count=1 -v -run TestRegisterControllerWithWrappers1

func TestRegisterControllerWithWrappers1(t *testing.T) {

	// init

	ci := &ControllerTestInit{t}

	r := newRouter()

	// indirect error
	if r == nil {
		t.Error("newRouter() returns nil")
		return
	}

	err := registerController(r, ci, false)

	if err == nil || !strings.HasSuffix(err.Error(), "ControllerTestInit.Init works fine!") {
		t.Error(err)
		return
	}

	// before
	testControllerInternal1(t, &ControllerTestBefore{t}, true, testControllerWithWrappers1Before)

	// after
	testControllerInternal1(t, &ControllerTestAfter{t}, true, testControllerWithWrappers1After)

	// all together
	cw := &ControllerTestWrappers{t, nil}

	testControllerInternal1(t, cw, true, testControllerWithWrappers1Wrappers)

	t.Logf("ControllerTestWrappers handlers seq: %v", cw.a)

	if wantSeq := []string{"init", "before", "handler", "after"}; !reflect.DeepEqual(cw.a, wantSeq) {
		t.Errorf("ControllerTestWrappers erroneous handlers seq: got %v, want %v", cw.a, wantSeq)
		return
	}
}
