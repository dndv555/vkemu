package api

import (
	"vkemu/internal/params"
	"vkemu/internal/vkscript"
)

func (s *Server) execute(c *call) (any, error) {
	code := c.p.Get("code")
	if code == "" {
		return nil, errParam
	}
	api := func(method string, args map[string]any) (any, error) {
		p := params.FromAny(args)
		p["method"] = method
		return s.invoke(c, method, p)
	}
	value, err := vkscript.Eval(code, api)
	if err != nil {
		return nil, errf(12, "Compile error: %s", err.Error())
	}
	return value, nil
}
