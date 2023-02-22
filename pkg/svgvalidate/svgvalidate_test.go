package svgvalidate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	svgs := []string{
		"<svg onload=prompt(/OPENBUGBOUNTY/)></svg>",
		"<svg><script>alert(1)</script></svg>",
		"<svg/onload=alert(1)><svg>",
		"<svg\nonload=alert(1)><svg>",
		`<svg onResize svg onResize="javascript:javascript:alert(1)"></svg onResize>`,
		"<svg	onload=alert(1)><svg>",
		"<svgonload=alert(1)><svg>"}

	svgValidator := NewValidator()

	for _, svg := range svgs {
		err := svgValidator.Validate([]byte(svg))
		require.Error(t, err)
	}

}

func TestComplex(t *testing.T) {
	svgs := []string{
		`<?xml version="1.0" encoding="UTF-8" standalone="no"?> 2<!DOCTYPE testingxxe [ <!ENTITY xml "eXtensible Markup Language"> ]> 3<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="200" height="200"> 4<image height="30" width="30" xlink:href="http://<usercontrolledserever>/" /> 5<text x="0" y="20" font-size="20">&xml;</text> 6</svg>`,
	}
	svgValidator := NewValidator()
	for _, svg := range svgs {
		err := svgValidator.Validate([]byte(svg))
		require.Error(t, err)
	}
}
