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
	"net/http"

	"fmt"
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

type RouterGroup = gin.IRoutes

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

func registerController(rg RouterGroup, instance interface{}, prependControllerEndpoint bool) error {

	const (
		errPrefix = "ginext.registerController error: "
	)

	v := reflect.ValueOf(instance)
	k := v.Kind()

	if k != reflect.Ptr && k != reflect.Interface {
		return fmt.Errorf(errPrefix+"wrong instance type: %[1]T (value %[1]v)", instance)
	}

	t, e := v.Type(), v.Elem()

	//TODO? check e is of struct kind

	// NOTE automagically counts all public methods of embedded types and overridings
	n := t.NumMethod()

	if n == 0 {
		return fmt.Errorf(errPrefix+"controller instance %v (%[1]T): methods not found", instance)
	}

	controllerName := e.Type().Name()

	var controllerEndpoint string

	if prependControllerEndpoint {
		if controllerEndpoint = strings.TrimPrefix(controllerName, ControllerPrefix); controllerEndpoint != "" {
			controllerEndpoint = strcase.KebabCase(controllerEndpoint)
		}
	}

	for i := 0; i < n; i++ {

		mi := t.Method(i)
		name := mi.Name

		if m, e := decodeControllerMethod(name); m != "" {

			if prependControllerEndpoint {
				e = controllerEndpoint + "/" + e
			}

			if appendTrailingSlash {
				e = e + "/"
			}

			// method's instance with stored receiver
			methodInstance := v.Method(i).Interface()

			// ATN! not .(HandlerFunc) :: panic: interface conversion: interface {} is func(*gin.Context), not gin.HandlerFunc [recovered]
			ginMethod, ok := methodInstance.(func(*gin.Context))

			if !ok {
				return fmt.Errorf(errPrefix+"controller instance %v of type %[1]T has action method %v with wrong signature %T", instance, name, methodInstance)
			}

			if m == methodActionNotch {
				rg.Any(e, ginMethod)
			} else {
				rg.Handle(m, e, ginMethod)
			}
		}
	}

	return nil
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
