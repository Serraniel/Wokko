// Command nutrition-data turns the USDA FoodData Central SR Legacy CSV download into the compact
// file Wokko ships for looking up nutrition values offline.
//
//	curl -LO https://fdc.nal.usda.gov/fdc-datasets/FoodData_Central_sr_legacy_food_csv_2018-04.zip
//	unzip FoodData_Central_sr_legacy_food_csv_2018-04.zip
//	go run ./cmd/nutrition-data -in FoodData_Central_sr_legacy_food_csv_2018-04
//
// SR Legacy is public domain (CC0) and final, so this only runs again for another dataset.
package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/serraniel/wokko/backend/internal/nutrition"
)

// Wokko's nutrient ids and the USDA ones they come from. Salt is sodium in mg × 2.5 / 1000.
var fromUSDA = map[string]int64{
	"1008": 1, // energy, kcal
	"1004": 2, // fat
	"1258": 3, // saturated fat
	"1005": 4, // carbohydrates
	"2000": 5, // sugars
	"1079": 6, // fiber
	"1003": 7, // protein
	"1093": 8, // sodium, mg → salt, g
}

func main() {
	in := flag.String("in", "", "folder with the unpacked SR Legacy CSV files")
	out := flag.String("out", "internal/nutrition/data/sr_legacy.json.gz", "file to write")
	flag.Parse()
	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	foods := map[int64]*nutrition.Item{}
	err := each(filepath.Join(in, "food.csv"), func(row map[string]string) error {
		id, err := strconv.ParseInt(row["fdc_id"], 10, 64)
		if err != nil {
			return err
		}
		foods[id] = &nutrition.Item{ID: id, Description: row["description"], Values: map[int64]float64{}}
		return nil
	})
	if err != nil {
		return err
	}
	err = each(filepath.Join(in, "food_nutrient.csv"), func(row map[string]string) error {
		n, ok := fromUSDA[row["nutrient_id"]]
		if !ok {
			return nil
		}
		id, err := strconv.ParseInt(row["fdc_id"], 10, 64)
		if err != nil {
			return err
		}
		f := foods[id]
		if f == nil {
			return nil
		}
		v, err := strconv.ParseFloat(row["amount"], 64)
		if err != nil {
			return err
		}
		if n == 8 {
			v = v * 2.5 / 1000
		}
		f.Values[n] = math.Round(v*1000) / 1000
		return nil
	})
	if err != nil {
		return err
	}
	list := make([]nutrition.Item, 0, len(foods))
	for _, f := range foods {
		if _, ok := f.Values[1]; ok {
			list = append(list, *f)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	zw, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(zw).Encode(list); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	fmt.Printf("%d foods written to %s\n", len(list), out)
	return nil
}

func each(path string, fn func(map[string]string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	head, err := r.Read()
	if err != nil {
		return err
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		row := make(map[string]string, len(head))
		for i, h := range head {
			row[h] = rec[i]
		}
		if err := fn(row); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
}
