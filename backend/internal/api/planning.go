package api

import (
	"net/http"

	"github.com/serraniel/wokko/backend/internal/app"
)

func decodeAnd[In, Out any](w http.ResponseWriter, r *http.Request, status int, fn func(In) (Out, error)) error {
	var in In
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := fn(in)
	if err != nil {
		return err
	}
	writeJSON(w, status, out)
	return nil
}

func (s *Server) planningRoutes(m *http.ServeMux, p string) {
	m.HandleFunc("GET "+p+"/meal-types", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListMealTypes(r.Context(), v)
		return reply(w, out, err)
	}))
	m.HandleFunc("POST "+p+"/meal-types", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.MealTypeInput) (app.MealType, error) { return s.svc.CreateMealType(r.Context(), v, in) })
	}))
	m.HandleFunc("PUT "+p+"/meal-types/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return decodeAnd(w, r, 200, func(in app.MealTypeInput) (app.MealType, error) { return s.svc.UpdateMealType(r.Context(), v, id, in) })
	}))
	m.HandleFunc("DELETE "+p+"/meal-types/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return s.withID(w, r, func(id int64) error { return s.svc.DeleteMealType(r.Context(), v, id) })
	}))

	m.HandleFunc("GET "+p+"/meal-plans", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListMealPlans(r.Context(), v, r.URL.Query().Get("from"), r.URL.Query().Get("to"), queryInt(r, "meal_type"))
		return reply(w, out, err)
	}))
	m.HandleFunc("POST "+p+"/meal-plans", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.MealPlanInput) (app.MealPlan, error) { return s.svc.CreateMealPlan(r.Context(), v, in) })
	}))
	m.HandleFunc("GET "+p+"/meal-plans/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		out, err := s.svc.GetMealPlan(r.Context(), v, id)
		return reply(w, out, err)
	}))
	m.HandleFunc("PUT "+p+"/meal-plans/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return decodeAnd(w, r, 200, func(in app.MealPlanInput) (app.MealPlan, error) { return s.svc.UpdateMealPlan(r.Context(), v, id, in) })
	}))
	m.HandleFunc("DELETE "+p+"/meal-plans/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return s.withID(w, r, func(id int64) error { return s.svc.DeleteMealPlan(r.Context(), v, id) })
	}))

	m.HandleFunc("GET "+p+"/shopping", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListShopping(r.Context(), v)
		return reply(w, out, err)
	}))
	m.HandleFunc("POST "+p+"/shopping", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.ShoppingItemInput) (app.ShoppingItem, error) {
			return s.svc.CreateShoppingItem(r.Context(), v, in)
		})
	}))
	m.HandleFunc("POST "+p+"/shopping/recipes", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.ShoppingAddRecipe) (app.ShoppingRecipe, error) {
			return s.svc.AddRecipeToShopping(r.Context(), v, in)
		})
	}))
	m.HandleFunc("POST "+p+"/shopping/check", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		var in app.ShoppingCheckInput
		if err := readJSON(r, &in); err != nil {
			return err
		}
		return done(w, s.svc.CheckShopping(r.Context(), v, in))
	}))
	m.HandleFunc("DELETE "+p+"/shopping/checked", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return done(w, s.svc.ClearCheckedShopping(r.Context(), v))
	}))
	m.HandleFunc("PUT "+p+"/shopping/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return decodeAnd(w, r, 200, func(in app.ShoppingItemInput) (app.ShoppingItem, error) {
			return s.svc.UpdateShoppingItem(r.Context(), v, id, in)
		})
	}))
	m.HandleFunc("DELETE "+p+"/shopping/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return s.withID(w, r, func(id int64) error { return s.svc.DeleteShoppingItem(r.Context(), v, id) })
	}))

	m.HandleFunc("GET "+p+"/categories", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListCategories(r.Context(), v)
		return reply(w, out, err)
	}))
	m.HandleFunc("POST "+p+"/categories", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.CategoryInput) (app.SupermarketCategory, error) {
			return s.svc.CreateCategory(r.Context(), v, in)
		})
	}))
	m.HandleFunc("PUT "+p+"/categories/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return decodeAnd(w, r, 200, func(in app.CategoryInput) (app.SupermarketCategory, error) {
			return s.svc.UpdateCategory(r.Context(), v, id, in)
		})
	}))
	m.HandleFunc("DELETE "+p+"/categories/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return s.withID(w, r, func(id int64) error { return s.svc.DeleteCategory(r.Context(), v, id) })
	}))
	m.HandleFunc("PUT "+p+"/foods/{id}/category", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		var in struct {
			CategoryID int64 `json:"category_id"`
		}
		if err := readJSON(r, &in); err != nil {
			return err
		}
		return s.withID(w, r, func(id int64) error { return s.svc.SetFoodCategory(r.Context(), v, id, in.CategoryID) })
	}))

	m.HandleFunc("GET "+p+"/supermarkets", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListSupermarkets(r.Context(), v)
		return reply(w, out, err)
	}))
	m.HandleFunc("POST "+p+"/supermarkets", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return decodeAnd(w, r, 201, func(in app.CategoryInput) (app.Supermarket, error) {
			return s.svc.CreateSupermarket(r.Context(), v, in)
		})
	}))
	m.HandleFunc("PUT "+p+"/supermarkets/{id}/order", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		var in struct {
			CategoryIDs []int64 `json:"category_ids"`
		}
		if err := readJSON(r, &in); err != nil {
			return err
		}
		out, err := s.svc.SetSupermarketOrder(r.Context(), v, id, in.CategoryIDs)
		return reply(w, out, err)
	}))
	m.HandleFunc("DELETE "+p+"/supermarkets/{id}", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		return s.withID(w, r, func(id int64) error { return s.svc.DeleteSupermarket(r.Context(), v, id) })
	}))

	m.HandleFunc("GET "+p+"/nutrients", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		out, err := s.svc.ListNutrients(r.Context(), v)
		return reply(w, out, err)
	}))
	m.HandleFunc("GET "+p+"/foods/{id}/nutrition", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		out, err := s.svc.GetFoodNutrition(r.Context(), id)
		return reply(w, out, err)
	}))
	m.HandleFunc("PUT "+p+"/foods/{id}/nutrition", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return decodeAnd(w, r, 200, func(in app.FoodNutrition) (app.FoodNutrition, error) {
			return s.svc.SetFoodNutrition(r.Context(), v, id, in)
		})
	}))
	m.HandleFunc("GET "+p+"/recipes/{id}/nutrition", s.user(func(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		out, err := s.svc.GetRecipeNutrition(r.Context(), v, id)
		return reply(w, out, err)
	}))
}
