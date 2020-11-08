package color

import (
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

// Stderr colored output for Windows
var Stderr = colorable.NewColorableStderr()

// Colors used in the cli
var (
	HiBlack  = color.New(color.FgHiBlack)
	HiBlue   = color.New(color.FgHiBlue)
	HiCyan   = color.New(color.FgHiCyan)
	HiGreen  = color.New(color.FgHiGreen)
	HiYellow = color.New(color.FgHiYellow)
	HiRed    = color.New(color.FgHiRed)
	// Bold     = color.New(color.Bold)

	SHiBlack  = HiBlack.Sprint
	SHiBlue   = HiBlue.Sprint
	SHiCyan   = HiCyan.Sprint
	SHiGreen  = HiGreen.Sprint
	SHiYellow = HiYellow.Sprint
	SHiRed    = HiRed.Sprint

	SHiBlackf  = HiBlack.Sprintf
	SHiBluef   = HiBlue.Sprintf
	SHiCyanf   = HiCyan.Sprintf
	SHiGreenf  = HiGreen.Sprintf
	SHiYellowf = HiYellow.Sprintf
	SHiRedf    = HiRed.Sprintf

	SHiBlackln  = HiBlack.Sprintln
	SHiBlueln   = HiBlue.Sprintln
	SHiCyanln   = HiCyan.Sprintln
	SHiGreenln  = HiGreen.Sprintln
	SHiYellowln = HiYellow.Sprintln
	SHiRedln    = HiRed.Sprintln
)
