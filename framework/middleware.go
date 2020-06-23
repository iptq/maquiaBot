package framework

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

const (
	MIDDLEWARE_RESPONSE_OK = iota
	MIDDLEWARE_RESPONSE_ERR
	MIDDLEWARE_RESPONSE_CONT
)

type Middleware interface {
	// Handle the input
	Handle(*CommandContext) int
}

type _Chain struct {
	Middleware
	inner Middleware
}

func Chain(inner Middleware, next Middleware) _Chain {
	return _Chain{Middleware: next, inner: inner}
}

func (c _Chain) Handle(ctx *CommandContext) int {
	res := c.inner.Handle(ctx)
	switch res {
	case MIDDLEWARE_RESPONSE_OK:
		fallthrough
	case MIDDLEWARE_RESPONSE_ERR:
		return res
	case MIDDLEWARE_RESPONSE_CONT:
		return c.Middleware.Handle(ctx)
	default:
		panic("bad response value")
	}
}

type _Wrap struct {
	Middleware
	regex    *regexp.Regexp
	helpFunc func(*discordgo.MessageEmbed)
}

func Wrap(inner Middleware, regex string) _Wrap {
	regexp := regexp.MustCompile(regex)
	w := _Wrap{Middleware: inner, regex: regexp}

	// look for a Help function
	if m, ok := lookForHelpFunc(reflect.ValueOf(inner), reflect.TypeOf(inner)); ok {
		fmt.Println("it's valid")
		w.helpFunc = m
	}
	return w
}

func lookForHelpFunc(val reflect.Value, ty reflect.Type) (func(*discordgo.MessageEmbed), bool) {
	// if it's an interface, unwrap
	for val.Kind() == reflect.Interface {
		val = val.Elem()
		ty = val.Type()
	}

	if m, ok := ty.MethodByName("Help"); ok {
		return func(embed *discordgo.MessageEmbed) {
			m.Func.Call([]reflect.Value{
				val,
				reflect.ValueOf(embed),
			})
		}, true
	}

	// if it's a struct
	if ty.Kind() == reflect.Struct {
		// try going over the fields
		for i := 0; i < ty.NumField(); i++ {
			field := val.Field(i)
			fieldTy := ty.Field(i)
			if fieldTy.Anonymous {
				return lookForHelpFunc(field, fieldTy.Type)
			}
		}
	}

	return func(*discordgo.MessageEmbed) {}, false
}

func (w _Wrap) Help(embed *discordgo.MessageEmbed) {
	if w.helpFunc != nil {
		w.helpFunc(embed)
	} else {
		embed.Description = "no help contents here"
	}
}

func (w _Wrap) Regex() *regexp.Regexp {
	return w.regex
}

func (w _Wrap) Handle(ctx *CommandContext) int {
	return w.Middleware.Handle(ctx)
}
