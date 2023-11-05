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
	"github.com/gin-gonic/gin"
	"github.com/stoewer/go-strcase"

	"fmt"
	"net/http"
	"reflect"
	"strings"
)

const (
	ControllerPrefix = "Controller"

	MethodActionPrefix    = "Action"
	methodActionPrefixLen = len(MethodActionPrefix)

	methodActionNotch = "*"
)

/*
type RouterGroup interface {
	Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	Any(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}
*/

type (
	RouterGroup = gin.IRoutes

	HandlerFunc          = func(ctx *gin.Context)
	ControllerInitMethod = func() error

	/* TODO
	// can't use IRoutes because of BasePath absence
	RouterGroup interface {
		Handle(string, string, ...gin.HandlerFunc) gin.IRoutes
		Any(string, ...gin.HandlerFunc) gin.IRoutes

		BasePath() string
	}

	ControllerInitMethod = func(absPath string) error
	*/
)

var (
	// RFC7231 Section 4.3
	// ATN! array, not slice!
	rfcHttpMethods = [...]string{
		http.MethodGet, http.MethodPost, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete, http.MethodConnect, http.MethodTrace,
	}

	appendTrailingSlash = false
)

func AppendTrailingSlash(on bool) {
	appendTrailingSlash = on
}

func AttachController(rg RouterGroup, instance interface{}) error {
	return registerController(rg, instance, true)
}

func EmbedController(rg RouterGroup, instance interface{}) error {
	return registerController(rg, instance, false)
}

const (
	errRegisterControllerPrefix = "ginext.registerController error: "
)

func registerController(rg RouterGroup, instance interface{}, prependControllerEndpoint bool) (err error) {

	v := reflect.ValueOf(instance)
	k := v.Kind()

	if k != reflect.Ptr && k != reflect.Interface {
		return fmt.Errorf(errRegisterControllerPrefix+"wrong instance type: %[1]T (value %[1]v)", instance)
	}

	t, e := v.Type(), v.Elem()

	//TODO? check e is of struct kind

	// NOTE automagically counts all public methods of embedded types and overridings
	n := t.NumMethod()

	if n == 0 {
		return fmt.Errorf(errRegisterControllerPrefix+"controller instance %v (%[1]T): methods not found", instance)
	}

	controllerName := e.Type().Name()

	var controllerEndpoint string

	if prependControllerEndpoint {
		if controllerEndpoint = strings.TrimPrefix(controllerName, ControllerPrefix); controllerEndpoint != "" {
			controllerEndpoint = strcase.KebabCase(controllerEndpoint)
		}
	}

	// Init, Before, After
	methodValue := v.MethodByName("Init")

	// SEE https://github.com/golang/go/issues/46320#issuecomment-1081940201
	if methodValue.IsValid() && !methodValue.IsNil() {

		// TODO absPath
		//absPath := joinPaths(rg.BasePath, controllerEndpoint)

		initInstance := methodValue.Interface()
		initMethod, ok := initInstance.(ControllerInitMethod)

		if !ok {
			return fmt.Errorf(errRegisterControllerPrefix+"controller instance %v of type %[1]T has Init method with wrong signature %T", instance, initInstance)
		}

		// early call before method registration
		if err = initMethod(); err != nil {
			return fmt.Errorf(errRegisterControllerPrefix+"controller instance %v of type %[1]T Init err: %w", instance, err)
		}
	}

	// equivalent of struct { Before, handler, After } as an array,
	// possibleHandlers[0] = Before
	// possibleHandlers[1] = methodHandler
	// possibleHandlers[2] = After
	var (
		possibleHandlers [3]gin.HandlerFunc
		actualHandlers   = possibleHandlers[:]
	)

	// Before

	if possibleHandlers[0], err = extractWrapperMethod(instance, &v, "Before"); err != nil {
		return err
	}

	// shift actualHandlers if no Before
	if possibleHandlers[0] == nil {
		actualHandlers = actualHandlers[1:]
	}

	// After

	if possibleHandlers[2], err = extractWrapperMethod(instance, &v, "After"); err != nil {
		return err
	}

	// shift actualHandlers if no After
	if possibleHandlers[2] == nil {
		actualHandlers = actualHandlers[:len(actualHandlers)-1]
	}

	// now actualHandlers slices non-nil wrapper methods + handler (possibleHandlers[1])

	for i := 0; i < n; i++ {

		mi := t.Method(i)
		name := mi.Name

		if m, e := decodeControllerMethod(name); m != "" {

			if prependControllerEndpoint {
				e = controllerEndpoint + "/" + e
			}

			if appendTrailingSlash && e[len(e)-1] != '/' { // avoid double trailing slash
				e = e + "/"
			}

			// method's instance with stored receiver
			methodInstance := v.Method(i).Interface()

			var ok bool

			// ATN! not gin.HandlerFunc :: panic: interface conversion: interface {} is func(*gin.Context), not gin.HandlerFunc [recovered]
			possibleHandlers[1], ok = methodInstance.(HandlerFunc)

			if !ok {
				return fmt.Errorf(errRegisterControllerPrefix+"controller instance %v of type %[1]T has action method %v with wrong signature %T", instance, name, methodInstance)
			}

			if m == methodActionNotch {
				rg.Any(e, actualHandlers...)
			} else {
				rg.Handle(m, e, actualHandlers...)
			}
		}
	}

	return nil
}

func extractWrapperMethod(instance interface{}, v *reflect.Value, method string) (gin.HandlerFunc, error) {

	methodValue := v.MethodByName(method)

	// SEE https://github.com/golang/go/issues/46320#issuecomment-1081940201
	if methodValue.IsValid() && !methodValue.IsNil() {

		methodInstance := methodValue.Interface()

		// ATN! not gin.HandlerFunc :: panic: interface conversion: interface {} is func(*gin.Context), not gin.HandlerFunc [recovered]
		if wrapperMethod, ok := methodInstance.(HandlerFunc); ok {
			return wrapperMethod, nil
		}

		// !ok => error wrong type
		return nil, fmt.Errorf(errRegisterControllerPrefix+"controller instance %v of type %[1]T has wrapper method %v with wrong signature %T", instance, method, methodInstance)
	}

	return nil, nil
}

func decodeControllerMethod(method string) (m, e string) {

	if strings.HasPrefix(method, MethodActionPrefix) {
		return methodActionNotch, strcase.KebabCase(method[methodActionPrefixLen:])
	}

	if m = httpMethod(method); m == "" {
		return "", ""
	}

	return m, strcase.KebabCase(method[len(m):])
}

func httpMethod(s string) string {

	s = strings.ToUpper(s)

	for i := 0; i < len(rfcHttpMethods); i++ {
		if m := rfcHttpMethods[i]; strings.HasPrefix(s, m) {
			return m
		}
	}

	return ""
}
