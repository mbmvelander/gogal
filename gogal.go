package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Measurement struct {
	Value float64
	SDev  float64
}

type Shear struct {
	One   Measurement
	Two   Measurement
	EMode Measurement
	BMode Measurement
}

type FlexionF struct {
	One   Measurement
	Two   Measurement
	EMode Measurement
	BMode Measurement
}

type FlexionG struct {
	One   Measurement
	Two   Measurement
	EMode Measurement
	BMode Measurement
}

type Lens struct {
	ID int
	X  float64
	Y  float64
}

type Source struct {
	ID       int
	X        float64
	Y        float64
	Shear    Shear
	FlexionF FlexionF
	FlexionG FlexionG
}

type GalGalData struct {
	Lens     Lens
	Source   Source
	DeltaTh  float64
	DeltaX  float64
	DeltaY  float64
	Phi      float64
	Shear    Shear
	FlexionF FlexionF
	FlexionG FlexionG
}

func (g GalGalData) String() string {
	return fmt.Sprintf("%d %d %g %g %g %g %d %d %g %g %g %g %g %g %g %g %g %g %g %g %g %g %g %g",
		g.Lens.ID, g.Source.ID, g.Lens.X, g.Lens.Y, g.Source.X, g.Source.Y, 0, 0, g.DeltaTh, g.DeltaX, g.DeltaY, g.Phi,
		g.Shear.EMode.Value, g.Shear.EMode.SDev, g.Shear.BMode.Value, g.Shear.BMode.SDev,
		g.FlexionF.EMode.Value, g.FlexionF.EMode.SDev, g.FlexionF.BMode.Value, g.FlexionF.BMode.SDev,
		g.FlexionG.EMode.Value, g.FlexionG.EMode.SDev, g.FlexionG.BMode.Value, g.FlexionG.BMode.SDev,
	)
}

func main() {
	fluxSeparator := 1011.830 //1000.0
	scanner := bufio.NewScanner(os.Stdin)
	lenses := []Lens{}
	sources := []Source{}
	for scanner.Scan() {
		line := scanner.Text()
		// Ignore comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		flux, err := getMeasurement(fields, "flux", 3, 4)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			continue
		}
		if flux.Value > fluxSeparator {
			lenses = append(lenses, getLensData(fields))
		} else {
			sources = append(sources, getSourceData(fields))
		}
	}
	fmt.Printf("count lens: %v; count source: %v\n", len(lenses), len(sources))
	printChan := make(chan GalGalData)
	defer close(printChan)
	var wg sync.WaitGroup
	for _, lens := range lenses {
		wg.Add(1)
		go func(lens Lens) {
			defer wg.Done()
			for _, source := range sources {
				if lens.ID == 184 && source.ID == 103 {
					fmt.Printf("LENS 184: %v\n", lens)
					fmt.Printf("SOURCE 103: %v\n", source)
				}
				deltaX := source.X - lens.X
				deltaY := source.Y - lens.Y
				deltaTh := math.Sqrt(deltaX*deltaX + deltaY*deltaY)
				if deltaTh >= 54838.0 || deltaTh <= 37.0 {
					continue
				}
				phi := math.Atan2(deltaY, deltaX)
				seVal := -math.Cos(2.*phi)*source.Shear.One.Value - math.Sin(2.*phi)*source.Shear.Two.Value
				sbVal := -math.Sin(2.*phi)*source.Shear.One.Value + math.Cos(2.*phi)*source.Shear.Two.Value
				feVal := -math.Cos(1.*phi)*source.FlexionF.One.Value - math.Sin(1.*phi)*source.FlexionF.Two.Value
				fbVal := -math.Sin(1.*phi)*source.FlexionF.One.Value + math.Cos(1.*phi)*source.FlexionF.Two.Value
				geVal := -math.Cos(3.*phi)*source.FlexionG.One.Value - math.Sin(3.*phi)*source.FlexionG.Two.Value
				gbVal := -math.Sin(3.*phi)*source.FlexionG.One.Value - math.Cos(3.*phi)*source.FlexionG.Two.Value
				feVal = feVal / 0.186
				fbVal = fbVal / 0.186
				geVal = geVal / 0.186
				gbVal = gbVal / 0.186
				se := Shear{EMode: Measurement{Value: seVal, SDev: source.Shear.One.SDev}, BMode: Measurement{Value: sbVal, SDev: source.Shear.Two.SDev}}
				fe := FlexionF{EMode: Measurement{Value: feVal, SDev: source.FlexionF.One.SDev}, BMode: Measurement{Value: fbVal, SDev: source.FlexionF.Two.SDev}}
				ge := FlexionG{EMode: Measurement{Value: geVal, SDev: source.FlexionG.One.SDev}, BMode: Measurement{Value: gbVal, SDev: source.FlexionG.Two.SDev}}
				printChan <- GalGalData{Lens: lens, Source: source, DeltaTh: deltaTh, DeltaX: deltaX, DeltaY: deltaY, Phi: phi, Shear: se, FlexionF: fe, FlexionG: ge}
			}
		}(lens)
	}
	go func() {
		for {
			data := <-printChan
			fmt.Printf("%v\n", data)
		}
	}()
	wg.Wait()
}

func getLensData(fields []string) Lens {
	x, y := getPosition(fields)
	id, _ := strconv.Atoi(fields[0])
	if fields[0] == "184" {
		fmt.Printf("AAAAAAAAAAAAAAAAAA %v\n", id)
	}
	return Lens{ID: id, X: x, Y: y}
}

func getSourceData(fields []string) Source {
	id, _ := strconv.Atoi(fields[0])
	x, y := getPosition(fields)
	shear := getShear(fields)
	flexionF := getFlexionF(fields)
	flexionG := getFlexionG(fields)
	return Source{ID: id, X: x, Y: y, Shear: shear, FlexionF: flexionF, FlexionG: flexionG}
}

func getMeasurement(fields []string, name string, valPos, errPos int) (Measurement, error) {
	value, err := strconv.ParseFloat(fields[valPos], 64)
	if err != nil {
		return Measurement{}, fmt.Errorf("object %s: could not parse `%s` as %s", fields[0], fields[valPos], name)
	}
	var sDev float64
	if errPos > 0 {
		var err error
		sDev, err = strconv.ParseFloat(fields[errPos], 64)
		if err != nil {
			return Measurement{}, fmt.Errorf("object %s: could not parse `%s` as %s error", fields[0], fields[errPos], name)
		}
	} else {
		sDev = 0.0
	}
	return Measurement{Value: value, SDev: sDev}, nil
}

func getPosition(fields []string) (float64, float64) {
	x, err := getMeasurement(fields, "x position", 1, -1)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return 0.0, 0.0
	}
	y, err := getMeasurement(fields, "y position", 2, -1)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return 0.0, 0.0
	}
	return x.Value, y.Value
}

func getShear(fields []string) Shear {
	s1, err := getMeasurement(fields, "shear 1", 13, 14)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return Shear{}
	}
	s2, err := getMeasurement(fields, "shear 2", 15, 16)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return Shear{}
	}
	return Shear{One: s1, Two: s2}
}

func getFlexionF(fields []string) FlexionF {
	f1, err := getMeasurement(fields, "flexion F 1", 17, 18)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return FlexionF{}
	}
	f2, err := getMeasurement(fields, "Flexion F 2", 19, 20)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return FlexionF{}
	}
	return FlexionF{One: f1, Two: f2}
}

func getFlexionG(fields []string) FlexionG {
	g1, err := getMeasurement(fields, "flexion G 1", 21, 22)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return FlexionG{}
	}
	g2, err := getMeasurement(fields, "Flexion G 2", 23, 24)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return FlexionG{}
	}
	return FlexionG{One: g1, Two: g2}
}
