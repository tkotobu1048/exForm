package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	"encoding/json"
	"os"
)

var (
	inputFile string

	minRate   int
	lineMin   int
	splitThls int

	hLines []XLine
	vLines []XLine

	colorModel color.Model
	upLimit    uint32

	tlfMargin int
	scaleX    int
	scaleY    int

	white color.RGBA
	red   color.RGBA

	adjThls int
)

type XLine struct {
	X         int
	Y         int
	Width     int
	Angle     int
	Value     []uint32
	SplitCnt  int
	EdgeCnt   int
	LastVal   uint32
	ChangeCnt int
}

type TlfLine struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Display     bool   `json:"display"`
	Description string `json:"description"`
	Style       struct {
		BorderColor string `json:"border-color"`
		BorderWidth int    `json:"border-width"`
		BorderStyle string `json:"border-style"`
	} `json:"style"`
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type Tlf struct {
	Version string    `json:"version"`
	Items   []TlfLine `json:"items"`
	State   struct {
		LayoutGuides []interface{} `json:"layout-guides"`
	} `json:"state"`
	Title  string `json:"title"`
	Report struct {
		PaperType   string `json:"paper-type"`
		Orientation string `json:"orientation"`
		Margin      []int  `json:"margin"`
	} `json:"report"`
}

func main() {
	flag.StringVar(&inputFile, "i", "", "input image file (PNG)")
	flag.Parse()

	inFile, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}

	defer inFile.Close()
	img, err := png.Decode(inFile)

	if err != nil {
		panic(err)
	}

	tlfMargin = 20

	var xMax int
	var yMax int
	orientation := ""
	if img.Bounds().Dy() >= img.Bounds().Dx() {
		orientation = "portrait"
		xMax = 555 + tlfMargin*2
		yMax = 800 + tlfMargin*2
	} else {
		orientation = "landscape"
		xMax = 800 + tlfMargin*2
		yMax = 555 + tlfMargin*2
	}
	scaleX = img.Bounds().Dx() * 100 / xMax
	scaleY = img.Bounds().Dy() * 100 / yMax

	fmt.Printf("scalex:%v scaley:%v\n", scaleX, scaleY)
	exLine(img)

	lines := make([]TlfLine, 0)
	scaledHL := make([]TlfLine, 0)
	scaledVL := make([]TlfLine, 0)

	var sLine *TlfLine

	// 水平線をスケーリングする
	for _, l := range hLines {
		sLine = new(TlfLine)

		sLine.X1 = l.X * 100 / scaleX
		sLine.X2 = (l.X+l.Len())*100/scaleX - 1
		sLine.Y1 = l.Y * 100 / scaleY
		sLine.Y2 = l.Y * 100 / scaleY

		scaledHL = append(scaledHL, *sLine)

	}

	// 垂直線をスケーリングする
	for _, l := range vLines {
		sLine = new(TlfLine)

		sLine.X1 = l.X * 100 / scaleX
		sLine.X2 = l.X * 100 / scaleX
		sLine.Y1 = l.Y * 100 / scaleY
		sLine.Y2 = (l.Y+l.Len())*100/scaleY - 1

		scaledVL = append(scaledVL, *sLine)
	}

	adjThls = 3

	// TLFに水平線を設定　および端点調整
	for _, hl := range scaledHL {
		sLine = new(TlfLine)
		setTLFLine(sLine)

		sLine.X1 = hl.X1
		sLine.Y1 = hl.Y1
		sLine.X2 = hl.X2
		sLine.Y2 = hl.Y2

		for _, vl := range scaledVL {
			adjustLen(&vl, sLine, true)
		}

		lines = append(lines, *sLine)
		fmt.Printf("HL x:%v y:%v \n", sLine.X1, sLine.Y1)
	}

	// TLFに垂直線を設定　および下端調整
	for _, sl := range scaledVL {
		sLine = new(TlfLine)
		setTLFLine(sLine)

		sLine.X1 = sl.X1
		sLine.Y1 = sl.Y1
		sLine.X2 = sl.X2
		sLine.Y2 = sl.Y2

		for _, hl := range scaledHL {
			adjustLen(&hl, sLine, false)
		}

		lines = append(lines, *sLine)
		fmt.Printf("VL x:%v y:%v \n", sLine.X1, sLine.Y1)
	}

	tlf := new(Tlf)

	// version
	tlf.Version = "0.10.0"

	// items
	tlf.Items = lines

	// state
	dummy := make([]interface{}, 0)
	tlf.State.LayoutGuides = dummy

	// report
	tlf.Report.PaperType = "A4"
	tlf.Report.Orientation = orientation
	margins := []int{20, 20, 20, 20}
	tlf.Report.Margin = margins

	jf, err := json.Marshal(tlf)
	os.WriteFile(inputFile+".tlf", jf, 0777)
}

func adjustLen(base, targ *TlfLine, targIsHLine bool) {
	if targIsHLine {
		difX1 := getAbsDif(base.X1, targ.X1)
		difX2 := getAbsDif(base.X2, targ.X2)

		if base.Y1-adjThls <= targ.Y1 && targ.Y1 <= base.Y2+adjThls {
			//fmt.Printf("baseY1:%v targY1:%v baseY2:%v\n", base.Y1, targ.Y1, base.Y2)
			if difX1 <= adjThls {
				targ.X1 = base.X1
			}
			if difX2 <= adjThls {
				targ.X2 = base.X1
			}
		}

	} else {
		difY1 := getAbsDif(base.Y1, targ.Y1)
		difY2 := getAbsDif(base.Y2, targ.Y2)

		if base.X1-adjThls <= targ.X1 && targ.X1 <= base.X2+adjThls {
			if difY1 <= adjThls {
				targ.Y1 = base.Y1
			}
			if difY2 <= adjThls {
				targ.Y2 = base.Y1
			}
		}
	}
}

func getAbsDif(val1, val2 int) int {
	return int(math.Abs(float64(val1) - float64(val2)))
}

func exLine(img image.Image) {
	minRate = 30

	// 最短の線の長さ(線の出力基準)算出
	lineMin = minRate * scaleX / 100

	splitThls = lineMin / 15

	hLines = make([]XLine, 0)
	vLines = make([]XLine, 0)

	colorModel = img.ColorModel()
	upLimit = uint32(240)

	fmt.Printf("Image Size: X:%v Y:%v\n", img.Bounds().Dx(), img.Bounds().Dy())

	var line *XLine

	// 水平線抽出
	line = nil
	for y := 0; y < img.Bounds().Dy(); y++ {

		for x := 0; x < img.Bounds().Dx(); x++ {

			v := getPointValue(img, x, y)

			if line == nil {
				if v < upLimit {
					line = newLine(x, y, v, 0)
				}
			} else {
				if y > 0 {
					line.chkEdge(getPointValue(img, x, y-1))
				} else {
					line.chkEdge(upLimit)
				}

				if !line.addValue(v) {
					if line.isWritable() {
						hLines = append(hLines, *line)
					}
					line = nil
				}
			}

		}
		line = nil
	}

	// 垂直線抽出
	line = nil
	for x := 0; x < img.Bounds().Dx(); x++ {

		for y := 0; y < img.Bounds().Dy(); y++ {

			v := getPointValue(img, x, y)

			if line == nil {
				if v < upLimit {
					line = newLine(x, y, v, 90)
				}
			} else {
				if x > 0 {
					line.chkEdge(getPointValue(img, x-1, y))
				} else {
					line.chkEdge(upLimit)
				}

				if !line.addValue(v) {
					if line.isWritable() {
						vLines = append(vLines, *line)
					}
					line = nil
				}
			}

		}
		line = nil
	}
}

func setTLFLine(tl *TlfLine) {
	tl.ID = ""
	tl.Type = "line"
	tl.Display = true
	tl.Description = ""
	tl.Style.BorderColor = "#000000"
	tl.Style.BorderStyle = "solid"
	tl.Style.BorderWidth = 1
}

func newLine(x, y int, v uint32, ang int) *XLine {
	val := make([]uint32, 0)
	xl := &XLine{
		X:         x,
		Y:         y,
		Width:     1,
		Angle:     ang,
		Value:     val,
		SplitCnt:  0,
		EdgeCnt:   0,
		LastVal:   upLimit,
		ChangeCnt: 0,
	}
	xl.addValue(v)
	return xl
}

func (xl *XLine) addValue(v uint32) bool {
	if v < upLimit {
		xl.Value = append(xl.Value, v)
		xl.SplitCnt = 0

		if v != xl.LastVal {
			xl.ChangeCnt += 1
		}
		xl.LastVal = v

		return true
	}

	if len(xl.Value) > 0 {
		xl.Value = append(xl.Value, v)
		xl.SplitCnt += 1

		if xl.SplitCnt > splitThls {
			if len(xl.Value) > 0 {
				return false
			}
		}
	} else {
		return true
	}

	return true
}
func (xl *XLine) chkEdge(v uint32) {
	if v >= upLimit {
		xl.EdgeCnt += 1
	}
}

func (xl *XLine) Len() int {
	return len(xl.Value) - xl.SplitCnt
}

func (xl *XLine) isWritable() bool {
	return xl.Len() > lineMin &&
		(xl.EdgeCnt > xl.Len()/3) &&
		xl.ChangeCnt < xl.Len()/3
}

func getPointValue(img image.Image, x, y int) uint32 {
	r, _, _, _ := img.At(x, y).RGBA()
	return uint32(r / 256)
}
