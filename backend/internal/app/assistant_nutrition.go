package app

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// Nutrient ids, as in the nutrients table.
const (
	nutrientEnergy  = 1
	nutrientFat     = 2
	nutrientSatFat  = 3
	nutrientCarbs   = 4
	nutrientSugar   = 5
	nutrientFiber   = 6
	nutrientProtein = 7
	nutrientSalt    = 8
)

// AssistantNutrition is what an assistant fills in, and what it couldn't.
type AssistantNutrition struct {
	FoodNutrition
	Filled  []int64 `json:"filled"`
	Skipped []int64 `json:"skipped"` // already had a value, which an assistant never changes
}

// FillFoodNutrition lets an assistant fill in a food's missing nutrition values. It never changes
// a value that exists, whether from the catalog or from a person. Values must be plausible and
// name their source, and they stay marked as unchecked until a person saves them. The person's
// choice about assistants adding catalog entries applies: never, ask first, or allow.
func (s *Service) FillFoodNutrition(ctx context.Context, v Viewer, id int64, values map[int64]float64, pieceGrams *float64, source string, confirmed bool) (AssistantNutrition, error) {
	mode := NewEntriesAsk
	if v.User != nil {
		mode = v.User.AssistantNewEntries
	}
	if mode == NewEntriesNever {
		return AssistantNutrition{}, fail(http.StatusForbidden, "assistant_catalog_not_allowed", "Nothing was saved. This person doesn't let assistants add to the catalog, so they enter nutrition values themselves.")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return AssistantNutrition{}, invalid("source_required", "Name where the values come from, such as the dataset entry from suggest_food_nutrition, a package label, or \"estimate\" if they are guessed.")
	}
	if len([]rune(source)) > 300 {
		return AssistantNutrition{}, invalid("too_long", "The source is too long")
	}
	var out AssistantNutrition
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		f, err := q.GetFood(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !s.canEditCatalog(v, f.CreatedBy.Int64) {
			return ErrForbidden
		}
		known, err := q.ListNutrients(ctx, v.locale())
		if err != nil {
			return err
		}
		valid := map[int64]bool{}
		for _, n := range known {
			valid[n.ID] = true
		}
		rows, err := q.ListFoodNutrients(ctx, []int64{id})
		if err != nil {
			return err
		}
		have := map[int64]float64{}
		for _, r := range rows {
			have[r.NutrientID] = r.Per100g
		}
		merged := map[int64]float64{}
		for n, x := range have {
			merged[n] = x
		}
		var filled, skipped []int64
		for n, x := range values {
			if !valid[n] {
				return invalid("unknown_nutrient", fmt.Sprintf("There is no nutrient %d", n))
			}
			if _, ok := have[n]; ok {
				skipped = append(skipped, n)
				continue
			}
			merged[n] = x
			filled = append(filled, n)
		}
		if err := plausible(merged); err != nil {
			return err
		}
		fillPiece := pieceGrams != nil && !f.PieceGrams.Valid
		if fillPiece && (*pieceGrams <= 0 || *pieceGrams > 5000) {
			return invalid("implausible_nutrition", "A piece should weigh between 0 and 5000 g")
		}
		if len(filled) == 0 && !fillPiece {
			return invalid("nothing_to_fill", "Nothing was saved: every value given already exists. Assistants only fill missing values. The person can change existing ones in Wokko.")
		}
		if mode == NewEntriesAsk && !confirmed {
			return invalid("assistant_nutrition_needs_confirmation", "Nothing was saved. Show the person these values and their source, and send the same call again with confirmed: true only after they agree.")
		}
		for _, n := range filled {
			if err := q.SetFoodNutrient(ctx, sqlcgen.SetFoodNutrientParams{FoodID: id, NutrientID: n, Per100g: values[n]}); err != nil {
				return err
			}
		}
		if fillPiece {
			if err := q.SetFoodPieceGrams(ctx, sqlcgen.SetFoodPieceGramsParams{PieceGrams: sql.NullFloat64{Float64: *pieceGrams, Valid: true}, ID: id}); err != nil {
				return err
			}
		}
		if f.NutritionSource != "" {
			source = f.NutritionSource + "; " + source
			if r := []rune(source); len(r) > 300 {
				source = string(r[:300])
			}
		}
		if err := q.SetFoodNutritionSource(ctx, sqlcgen.SetFoodNutritionSourceParams{NutritionSource: source, ID: id}); err != nil {
			return err
		}
		if err := q.SetFoodNutritionUnchecked(ctx, sqlcgen.SetFoodNutritionUncheckedParams{NutritionUnchecked: 1, ID: id}); err != nil {
			return err
		}
		sort.Slice(filled, func(i, j int) bool { return filled[i] < filled[j] })
		sort.Slice(skipped, func(i, j int) bool { return skipped[i] < skipped[j] })
		out.Filled, out.Skipped = filled, skipped
		return nil
	})
	if err != nil {
		return AssistantNutrition{}, err
	}
	n, err := s.GetFoodNutrition(ctx, id)
	if err != nil {
		return AssistantNutrition{}, err
	}
	out.FoodNutrition = n
	if out.Filled == nil {
		out.Filled = []int64{}
	}
	if out.Skipped == nil {
		out.Skipped = []int64{}
	}
	return out, nil
}

// plausible refuses values per 100 g that can't be right, such as more than 100 g of fat or
// energy far from what protein, carbohydrates and fat add up to.
func plausible(v map[int64]float64) error {
	bad := func(msg string) error { return invalid("implausible_nutrition", "Nothing was saved: "+msg) }
	for n, x := range v {
		if x < 0 || math.IsNaN(x) || math.IsInf(x, 0) {
			return bad("values can't be negative")
		}
		if n == nutrientEnergy && x > 900 {
			return bad("no food has more than 900 kcal per 100 g")
		}
		if n != nutrientEnergy && x > 100 {
			return bad("no part of 100 g can weigh more than 100 g")
		}
	}
	const slack = 2.0 // rounding in labels and datasets
	if v[nutrientFat]+v[nutrientCarbs]+v[nutrientProtein]+v[nutrientFiber] > 100+slack {
		return bad("fat, carbohydrates, protein and fibre add up to more than 100 g")
	}
	if v[nutrientSatFat] > v[nutrientFat]+slack && hasAll(v, nutrientSatFat, nutrientFat) {
		return bad("saturated fat is more than fat")
	}
	if v[nutrientSugar] > v[nutrientCarbs]+slack && hasAll(v, nutrientSugar, nutrientCarbs) {
		return bad("sugar is more than carbohydrates")
	}
	if hasAll(v, nutrientEnergy, nutrientFat, nutrientCarbs, nutrientProtein) {
		expected := 4*v[nutrientProtein] + 4*v[nutrientCarbs] + 9*v[nutrientFat]
		if math.Abs(v[nutrientEnergy]-expected) > math.Max(40, 0.3*expected) {
			return bad(fmt.Sprintf("%.0f kcal doesn't fit protein, carbohydrates and fat, which give about %.0f kcal", v[nutrientEnergy], expected))
		}
	}
	return nil
}

func hasAll(v map[int64]float64, ids ...int64) bool {
	for _, id := range ids {
		if _, ok := v[id]; !ok {
			return false
		}
	}
	return true
}
