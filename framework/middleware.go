package framework

import (
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
	if cmd, ok := inner.(Command); ok {
		w.helpFunc = cmd.Help
	}
	return w
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
