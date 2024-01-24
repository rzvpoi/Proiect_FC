package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/gorilla/mux"
)

// ViewData is a struct to hold data to be displayed in the template

type InputEntry struct {
	Nume   string
	D      float64
	Miu    float64
	Lambda float64
}

type Inputs struct {
	Entries []InputEntry
}

type Results struct {
	P          []float64
	Theta      []float64
	Ps         []float64
	Pb         []float64
	Entries    Inputs
	Extra_vals []float64
}

func main() {
	r := mux.NewRouter()

	r.Headers("X-Content-Type-Options")
	// Define routes
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/submit", submitHandler).Methods("POST")

	// Serve static files (CSS, JS, etc.)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server started. Web address at localhost:8089")
	http.Handle("/", r)
	http.ListenAndServe(":8089", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Render the HTML template
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Provide initial empty data to the template
	data := InputEntry{}
	tmpl.Execute(w, data)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get the username and additional inputs from the form
	names := r.Form["nume"]
	ds := r.Form["d"]
	mius := r.Form["miu"]
	lambdas := r.Form["lambda"]

	theta_i, _ := strconv.ParseFloat(r.Form.Get("theta_i"), 64)
	theta_e, _ := strconv.ParseFloat(r.Form.Get("theta_e"), 64)

	phi_i, _ := strconv.ParseFloat(r.Form.Get("phi_i"), 64)
	phi_e, _ := strconv.ParseFloat(r.Form.Get("phi_e"), 64)

	if len(names) != len(ds) || len(ds) != len(mius) || len(mius) != len(lambdas) {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	var inputs Inputs

	// Iterate over the slices and append values to the struct
	for i := 0; i < len(names); i++ {
		d, _ := strconv.ParseFloat(ds[i], 64)
		miu, _ := strconv.ParseFloat(mius[i], 64)
		lambda, _ := strconv.ParseFloat(lambdas[i], 64)
		entry := InputEntry{
			Nume:   names[i],
			D:      d,
			Miu:    miu,
			Lambda: lambda,
		}
		inputs.Entries = append(inputs.Entries, entry)
	}

	vars := inputs.Entries

	// calcul a)
	Ps_thetaI := 610.5 * math.Pow(math.E, (17.269*theta_i)/(237.3+theta_i))
	Ps_thetaE := 610.5 * math.Pow(math.E, (21.875*theta_e)/(265.5+theta_e))

	Pi := (phi_i * Ps_thetaI) / 100
	Pe := (phi_e * Ps_thetaE) / 100

	var rvs []float64

	for i := 0; i < len(names); i++ {
		rvs = append(rvs, 50*math.Pow10(8)*vars[i].D*vars[i].Miu)
	}

	var Rv float64

	for i := 0; i < len(rvs); i++ {
		Rv += rvs[i]
	}

	var P []float64
	P = append(P, Pi)

	for i := 0; i < len(rvs)-1; i++ {
		sum_Rv := float64(0)
		for j := 0; j < i+1; j++ {
			sum_Rv += rvs[j]
		}
		P = append(P, Pi-(sum_Rv/Rv)*(Pi-Pe))
	}
	P = append(P, Pe)

	// calcul b)
	Rsi := 0.125 // m^2K/W
	Rse := 0.042 // m^2K/W

	var R []float64
	var RT float64

	for i := 0; i < len(names); i++ {
		R = append(R, (vars[i].D/100)/vars[i].Lambda)
		RT += R[i]
	}
	RT = RT + Rsi + Rse

	theta_si := theta_i - (Rsi/RT)*(theta_i-theta_e)

	var thetas []float64

	for i := 0; i < len(names); i++ {
		sum_R := Rsi

		for j := 0; j < i+1; j++ {
			sum_R += R[j]
		}

		thetas = append(thetas, theta_i-(sum_R)/RT*(theta_i-theta_e))
	}

	theta_se := thetas[len(names)-1]

	// calcul c)
	var Ps []float64

	Ps_thetaSi := 610.5 * math.Pow(math.E, (17.269*theta_si)/(237.3+theta_si))
	Ps = append(Ps, Ps_thetaSi)

	for i := 0; i < len(names)-1; i++ {
		Ps = append(Ps, 610.5*math.Pow(math.E, (17.269*thetas[i])/(237.3+thetas[i])))
	}

	Ps_thetaSe := 610.5 * math.Pow(math.E, (17.269*theta_se)/(237.3+theta_se))
	Ps = append(Ps, Ps_thetaSe)

	fmt.Println("Ps:", Ps)
	fmt.Println("P:", P)

	max_axis := math.Floor(Ps[0]) + 200
	var parallelAxisList = []opts.ParallelAxis{
		{Dim: 0, Name: "Pi", Max: max_axis, NameLocation: "start"},
	}

	for i := 0; i < len(names)-1; i++ {
		axis := []opts.ParallelAxis{
			{Dim: i + 1, Name: "P" + strconv.Itoa(i+1), Max: max_axis},
		}
		parallelAxisList = append(parallelAxisList, axis...)
	}

	axis := []opts.ParallelAxis{
		{Dim: len(Ps), Name: "Pe", Max: max_axis},
	}
	parallelAxisList = append(parallelAxisList, axis...)

	plot := charts.NewParallel()
	plot.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title: "Plot 1",
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithParallelAxisList(parallelAxisList),
	)
	var parallelDatPs []interface{}
	for _, v := range Ps {
		parallelDatPs = append(parallelDatPs, v)
	}

	var parallelDatP []interface{}
	for _, v := range P {
		parallelDatP = append(parallelDatP, v)
	}

	plot.AddSeries("Ps(x)", generateParallelData(parallelDatPs)).
		AddSeries("P(x)", generateParallelData(parallelDatP))

	f, _ := os.Create("static/plot1.html")
	plot.Render(f)

	var Pb []float64 = nil
	var db float64 = 0.2
	var miub float64 = 0.1
	extra_vals := []float64{db, miub}
	isIntersected := false
	for i := 0; i < len(Ps); i++ {
		if Ps[i] < P[i] {
			isIntersected = true
		}
	}

	if isIntersected {

		Rvb := 50 * math.Pow10(8) * (db / 1000) * miub

		rvsb := rvs

		rvsb = insert(rvsb, 1, Rvb)

		Rv += Rvb

		for i := 0; i < len(rvsb); i++ {
			var sum_Rv float64 = 0
			for j := 0; j < i+1; j++ {
				sum_Rv += rvsb[j]
			}
			Pb = append(Pb, Pi-(sum_Rv/Rv)*(Pi-Pe))
		}

		Pb = insert(Pb, 0, Pi)

		fmt.Println("Pb", Pb)

		max_axis := math.Floor(Ps[0]) + 200
		var parallelAxisList = []opts.ParallelAxis{
			{Dim: 0, Name: "Pi", Max: max_axis, NameLocation: "start"},
		}

		for i := 0; i < len(names); i++ {
			axis := []opts.ParallelAxis{
				{Dim: i + 1, Name: "P" + strconv.Itoa(i+1), Max: max_axis},
			}
			parallelAxisList = append(parallelAxisList, axis...)
		}

		axis := []opts.ParallelAxis{
			{Dim: len(Ps), Name: "Pe", Max: max_axis},
		}
		parallelAxisList = append(parallelAxisList, axis...)

		plot := charts.NewParallel()
		plot.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
			charts.WithTitleOpts(opts.Title{
				Title: "Plot 2",
			}),
			charts.WithLegendOpts(opts.Legend{Show: true}),
			charts.WithParallelAxisList(parallelAxisList),
		)
		var parallelDatPs []interface{}
		for _, v := range Ps {
			parallelDatPs = append(parallelDatPs, v)
		}

		var parallelDatP []interface{}
		for _, v := range Pb {
			parallelDatP = append(parallelDatP, v)
		}

		plot.AddSeries("Ps(x)", generateParallelData(parallelDatPs)).
			AddSeries("P(x)", generateParallelData(parallelDatP))

		f, _ := os.Create("static/plot2.html")
		plot.Render(f)

	}

	tmpl, err := template.ParseFiles("result.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	thetas_copy := thetas

	thetas_copy = insert(thetas_copy, 0, theta_si)

	isIntersected = false

	data := Results{
		P,
		thetas_copy,
		Ps,
		Pb,
		inputs,
		extra_vals,
	}
	tmpl.Execute(w, data)

}

func generateParallelData(data []interface{}) []opts.ParallelData {
	items := make([]opts.ParallelData, 0)
	for i := 0; i < len(data); i++ {
		items = append(items, opts.ParallelData{Value: data})
	}
	return items
}

func insert(a []float64, index int, value float64) []float64 {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
}
