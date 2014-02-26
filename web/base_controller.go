// The Base Controller package provides common functionality for all controllers
package web

import (
	"fmt"
	"reflect"

	aErrors "github.com/ArdanStudios/go-common/errors"
	"github.com/ArdanStudios/go-common/helper"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/validation"
	"github.com/goinggo/tracelog"
)

type (
	// BaseController provides access to common controller
	BaseController struct {
		beego.Controller
	}

	// MessageResponse provides the document structure for sending
	// a list of messages
	MessageResponse struct {
		Messages []string
	}
)

const (
	CACHE_CONTROL_HEADER = "Cache-control"
)

// CacheOutput outputs the cache control headrer for seconds passed in
func (this *BaseController) CacheOutput(seconds int64) {
	this.Ctx.Output.Header(CACHE_CONTROL_HEADER, fmt.Sprintf("private, must-revalidate, max-age=%d", seconds))
}

// ServeBlankModel serves an empty key/value pair map as Json
func (this *BaseController) ServeBlankModel() {
	this.Data["json"] = map[string]string{}
	this.ServeJson()
}

// ServeBlankModelList serves an empty slice of key/value pair maps as Json
func (this *BaseController) ServeBlankModelList() {
	this.Data["json"] = []map[string]string{}
	this.ServeJson()
}

// ServeJsonModel marshals the specified object as JSON
func (this *BaseController) ServeJsonModel(obj interface{}) {
	this.ServeJsonWithCache(obj, 0)
}

// ServeJsonWithCache marshals the specified object as JSON specifying cache time
func (this *BaseController) ServeJsonWithCache(obj interface{}, secondsToCache int64) {
	if secondsToCache > 0 {
		this.CacheOutput(secondsToCache)
	}

	this.Data["json"] = obj
	this.ServeJson()
}

// ServeUnAuthorized returns an Unauthorized error
func (this *BaseController) ServeUnAuthorized() {
	tracelog.INFO("BaseController", "ServeUnAuthorized", "UnAuthorized, Exiting")

	this.ServeMessageWithStatus(aErrors.UNAUTHORIZED_ERROR_CODE, aErrors.UNAUTHORIZED_ERROR_MSG)
	return
}

// ServeValidationError returns a Validation Error's list of messages with a validation err code.
func (this *BaseController) ServeValidationError() {
	this.Ctx.Output.SetStatus(aErrors.VALIDATION_ERROR_CODE)

	msgs := MessageResponse{}
	msgs.Messages = []string{aErrors.VALIDATION_ERROR_MSG}
	this.Data["json"] = &msgs
	this.ServeJson()
}

// ServeValidationErrors returns a Validation Error's list of messages with a validation err code.
func (this *BaseController) ServeValidationErrors(validationErrors []*validation.ValidationError) {
	this.Ctx.Output.SetStatus(aErrors.VALIDATION_ERROR_CODE)

	response := make([]string, len(validationErrors))
	for index, validationError := range validationErrors {
		response[index] = fmt.Sprintf("%s: %s", validationError.Field, validationError.String())
	}

	msgs := MessageResponse{}
	msgs.Messages = response
	this.Data["json"] = &msgs
	this.ServeJson()
}

// ServeError serves a error interface object.
func (this *BaseController) ServeError(err error) {
	tracelog.INFO("BaseController", "ServeError", "Application Error, Exiting")

	switch e := err.(type) {
	case *aErrors.AppError:

		if e.ErrorCode() != 0 {
			this.ServeMessageWithStatus(e.ErrorCode(), e.Error())

		} else {
			this.ServeMessageWithStatus(aErrors.APP_ERROR_CODE, e.Error())
		}

	default:
		this.ServeMessageWithStatus(aErrors.APP_ERROR_CODE, aErrors.APP_ERROR_MSG)
	}

	return
}

// ServeErrorResponse serves an error interface object
func (this *BaseController) ServeErrorResponse(err error) {
	tracelog.INFO("BaseController", "ServeErrorResponse", "Application Error, Exiting : %s", err)

	switch e := err.(type) {
	case *aErrors.AppError:
		if e.ErrorCode() != 0 {
			this.ServeMessageWithStatus(e.ErrorCode(), e.Error())
			return
		}

		this.ServeMessageWithStatus(aErrors.APP_ERROR_CODE, e.Error())
		return

	default:
		this.ServeMessageWithStatus(aErrors.APP_ERROR_CODE, err.Error())
		return
	}
}

// ServeAppError serves a generic application error
func (this *BaseController) ServeAppError() {
	tracelog.INFO("BaseController", "ServeAppError", "Application Error, Exiting")

	this.ServeMessageWithStatus(aErrors.APP_ERROR_CODE, aErrors.APP_ERROR_MSG)
	return
}

// ServeMessageWithStatus serves a HTTP status and message
func (this *BaseController) ServeMessageWithStatus(status int, msg string) {
	this.ServeMessagesWithStatus(status, []string{msg})
}

// ServeMessageWithStatus serves a HTTP status and messages
func (this *BaseController) ServeMessagesWithStatus(status int, msgs []string) {
	this.Ctx.Output.SetStatus(status)
	response := MessageResponse{Messages: msgs}
	this.Data["json"] = &response
	this.ServeJson()
}

// ParseAndValidate is used to parse any form and query parameters from the request and validate the values
func (this *BaseController) ParseAndValidate(params interface{}) bool {
	err := this.ParseForm(params)
	if err != nil {
		this.ServeMessageWithStatus(aErrors.VALIDATION_ERROR_CODE, aErrors.VALIDATION_ERROR_MSG)
		return false
	}

	valid := validation.Validation{}
	ok, err := valid.Valid(params)
	if err != nil {
		this.ServeMessageWithStatus(aErrors.VALIDATION_ERROR_CODE, aErrors.VALIDATION_ERROR_MSG)
		return false
	}

	if ok == false {
		// Build a map of the error messages
		messages2 := map[string]string{}
		val := reflect.ValueOf(params).Elem()
		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			tag := typeField.Tag
			tagValue := tag.Get("error")
			messages2[typeField.Name] = tagValue
		}

		// Build the error response
		errors := []string{}
		for _, err := range valid.Errors {
			message, ok := messages2[err.Field]
			if ok == true {
				errors = append(errors, message)
			} else {
				errors = append(errors, err.Message)
			}
		}

		this.ServeMessagesWithStatus(aErrors.VALIDATION_ERROR_CODE, errors)
		return false
	}

	return true
}

// CatchPanic is used to stop and process panics before they reach the Go runtime
func (this *BaseController) CatchPanic(err *error, UUID string, functionName string) {
	if helper.CatchPanic(err, UUID, functionName) {
		this.ServeAppError()
	}
}
