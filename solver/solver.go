package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/rdarder/guillotine"
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"
)

type Solution struct {
	Spec   *guillotine.CutSpec
	Layout *guillotine.LayoutTree
}

func main() {
	var nboards = flag.Int("nboards", 8, "Number of boards")
	var population = flag.Int("population", 300, "Population size")
	var tsize = flag.Int("tsize", 5, "Tournament size")
	var eliteSize = flag.Int("eliteSize", 10, "Elite size")
	var psel = flag.Float64("psel", 0.8, "Tournament selection probability")
	var cx = flag.String("crossover", "uniform", "Crossover strategy")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	var weightMutateMean = flag.Float64("weightMutateMean", 10,
		"Mean number of gene weights to be mutated on each individual")
	var configMutateMean = flag.Float64("configMutateMean", 10,
		"Mean number of pick configs to be mutated on each individual")
	var generations = flag.Int("generations", 1000, "Number of generations")
	var seed = flag.Int64("seed", time.Now().Unix(), "Random seed for repeatable runs")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var crossover guillotine.Crossover
	switch *cx {
	case "uniform":
		crossover = guillotine.UniformCrossover
	case "onepoint":
		crossover = guillotine.OnePointCrossover
	case "twopoint":
		crossover = guillotine.TwoPointCrossover
	default:
		panic("Invalid option for crossover")
	}

	r := rand.New(rand.NewSource(*seed))

	spec := &guillotine.CutSpec{}
	spec.MaxWidth = 500
	spec.Add(200, 100).
		Add(150, 100).
		Add(50, 50).
		Add(50, 70).
		Add(150, 100).
		Add(50, 50).
		Add(50, 70).Add(123, 231)

	ga := &guillotine.GeneticAlgorithm{
		Spec:      spec,
		Evaluator: (*guillotine.LayoutTree).Area,
		Mutator: guillotine.CompoundWeightConfigMutator{
			Weight: guillotine.NormalWeightMutator{
				Mean:   *weightMutateMean,
				StdDev: *weightMutateMean / 5,
			},
			Config: guillotine.NormalConfigMutator{
				Mean:   *configMutateMean,
				StdDev: *configMutateMean / 5,
			},
		}.Mutate,
		Breeder:         crossover,
		SelectorBuilder: guillotine.NewTournamentSelectorBuilder(*tsize, float32(*psel), r, true),
		R:               r,
		EliteSize:       uint(*eliteSize),
	}
	pop := guillotine.NewRandomPopulation(uint16(*nboards), uint(*population), r)
	rankedPop := ga.Evaluate(pop)
	for i := 1; i < *generations; i++ {
		pop = ga.Next(rankedPop)
		rankedPop = ga.Evaluate(pop)
	}

	bestLayout := guillotine.GetPhenotype(spec, rankedPop.Pop[0])
	drawer := guillotine.NewDrawer(bestLayout)
	drawing := drawer.Draw()
	b, err := json.Marshal(drawing)
	if err != nil {
		log.Fatal("error:", err)
	}
	os.Stdout.Write(b)

	fmt.Printf("\nWaste: %v\n", 100*float64(drawing.Sheet.Width*drawing.Sheet.Height-spec.TotalArea)/float64(spec.TotalArea))

	dest := image.NewRGBA(image.Rect(0, 0, int(drawing.Sheet.Width), int(drawing.Sheet.Height)))
	gc := draw2dimg.NewGraphicContext(dest)

	// Set some properties
	gc.SetFillColor(color.RGBA{0xff, 0xff, 0xff, 0xff})
	//gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
	gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})

	gc.SetLineWidth(5)
	gc.DPI *= 10

	for _, rect := range drawing.Boxes {
		gc.MoveTo(float64(rect.X), float64(rect.Y))
		gc.LineTo(float64(rect.X+rect.Width), float64(rect.Y))
		gc.MoveTo(float64(rect.X+rect.Width), float64(rect.Y))
		gc.LineTo(float64(rect.X+rect.Width), float64(rect.Y+rect.Height))
		gc.MoveTo(float64(rect.X+rect.Width), float64(rect.Y+rect.Height))
		gc.LineTo(float64(rect.X), float64(rect.Y+rect.Height))
		gc.MoveTo(float64(rect.X), float64(rect.Y+rect.Height))
		gc.LineTo(float64(rect.X), float64(rect.Y))
		gc.Close()
		gc.FillStroke()
	}

	// Save to file
	draw2dimg.SaveToPngFile("output.png", dest)
}
