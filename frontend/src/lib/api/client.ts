const API_BASE = import.meta.env.PUBLIC_API_URL || '/api';

interface RequestOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { method = 'GET', body, token } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  } else if (typeof localStorage !== 'undefined') {
    const stored = localStorage.getItem('access_token');
    if (stored) headers['Authorization'] = `Bearer ${stored}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    credentials: 'include',
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  // Auth
  setup: (data: { username: string; password: string; display_name?: string }) =>
    apiRequest<{ access_token: string; user: User }>('/auth/setup', { method: 'POST', body: data }),
  login: (data: { username: string; password: string }) =>
    apiRequest<{ access_token: string; user: User }>('/auth/login', { method: 'POST', body: data }),
  register: (data: { username: string; password: string; email?: string; invite_token?: string }) =>
    apiRequest<{ access_token: string; user: User }>('/auth/register', { method: 'POST', body: data }),
  refresh: () =>
    apiRequest<{ access_token: string }>('/auth/refresh', { method: 'POST' }),
  logout: () =>
    apiRequest<void>('/auth/logout', { method: 'POST' }),
  me: () =>
    apiRequest<User>('/auth/me'),

  // Recipes
  listRecipes: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams(params as Record<string, string>).toString();
    return apiRequest<Recipe[]>(`/recipes${q ? '?' + q : ''}`);
  },
  createRecipe: (data: Partial<Recipe>) =>
    apiRequest<Recipe>('/recipes', { method: 'POST', body: data }),
  getRecipe: (id: string) =>
    apiRequest<RecipeDetail>(`/recipes/${id}`),
  updateRecipe: (id: string, data: Partial<Recipe>) =>
    apiRequest<Recipe>(`/recipes/${id}`, { method: 'PUT', body: data }),
  deleteRecipe: (id: string) =>
    apiRequest<void>(`/recipes/${id}`, { method: 'DELETE' }),
  uploadRecipePhoto: (id: string, file: File) => {
    const form = new FormData();
    form.append('photo', file);
    return fetch(`${API_BASE}/recipes/${id}/photo`, {
      method: 'POST',
      credentials: 'include',
      headers: { Authorization: `Bearer ${localStorage.getItem('access_token') || ''}` },
      body: form,
    }).then(r => r.json());
  },
  createShareLink: (recipeId: string, data?: { expires_in_days?: number }) =>
    apiRequest<{ token: string; url: string }>(`/recipes/${recipeId}/share`, { method: 'POST', body: data }),

  // Recipe steps
  listSteps: (recipeId: string) =>
    apiRequest<RecipeStep[]>(`/recipes/${recipeId}/steps`),
  createStep: (recipeId: string, data: Partial<RecipeStep>) =>
    apiRequest<RecipeStep>(`/recipes/${recipeId}/steps`, { method: 'POST', body: data }),
  updateStep: (recipeId: string, stepId: string, data: Partial<RecipeStep>) =>
    apiRequest<RecipeStep>(`/recipes/${recipeId}/steps/${stepId}`, { method: 'PUT', body: data }),
  deleteStep: (recipeId: string, stepId: string) =>
    apiRequest<void>(`/recipes/${recipeId}/steps/${stepId}`, { method: 'DELETE' }),
  addStepIngredient: (recipeId: string, stepId: string, data: Partial<StepIngredient>) =>
    apiRequest<StepIngredient>(`/recipes/${recipeId}/steps/${stepId}/ingredients`, { method: 'POST', body: data }),
  removeStepIngredient: (recipeId: string, stepId: string, ingId: string) =>
    apiRequest<void>(`/recipes/${recipeId}/steps/${stepId}/ingredients/${ingId}`, { method: 'DELETE' }),

  // Ingredients
  searchIngredients: (q: string, limit = 10) =>
    apiRequest<Ingredient[]>(`/ingredients?search=${encodeURIComponent(q)}&limit=${limit}`),
  createIngredient: (data: Partial<Ingredient>) =>
    apiRequest<Ingredient>('/ingredients', { method: 'POST', body: data }),

  // Units
  searchUnits: (q: string, limit = 10) =>
    apiRequest<Unit[]>(`/units?search=${encodeURIComponent(q)}&limit=${limit}`),
  createUnit: (data: Partial<Unit>) =>
    apiRequest<Unit>('/units', { method: 'POST', body: data }),

  // Public
  listPublicRecipes: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams(params as Record<string, string>).toString();
    return apiRequest<Recipe[]>(`/public/recipes${q ? '?' + q : ''}`);
  },
  getPublicRecipe: (id: string) =>
    apiRequest<RecipeDetail>(`/public/recipes/${id}`),
  getSharedRecipes: (token: string) =>
    apiRequest<Recipe[]>(`/public/share/${token}`),

  // Admin
  listUsers: () => apiRequest<User[]>('/admin/users'),
  deleteUser: (id: string) => apiRequest<void>(`/admin/users/${id}`, { method: 'DELETE' }),
  createInvite: (data?: { max_uses?: number; expires_in_days?: number }) =>
    apiRequest<InviteLink>('/admin/invites', { method: 'POST', body: data }),
  listInvites: () => apiRequest<InviteLink[]>('/admin/invites'),
  deleteInvite: (id: string) => apiRequest<void>(`/admin/invites/${id}`, { method: 'DELETE' }),
  getSettings: () => apiRequest<SystemSettings>('/admin/settings'),
  updateSettings: (data: Partial<SystemSettings>) =>
    apiRequest<void>('/admin/settings', { method: 'PUT', body: data }),
};

// Types
export interface User {
  id: string;
  username: string;
  email?: string;
  display_name?: string;
  role: 'admin' | 'user';
  created_at: string;
}

export interface Recipe {
  id: string;
  author_id: string;
  name: string;
  description?: string;
  photo_path?: string;
  work_time_min?: number;
  servings?: number;
  is_public: boolean;
  source_url?: string;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

export interface RecipeStep {
  id: string;
  recipe_id: string;
  step_order: number;
  description: string;
  duration_min?: number;
  ingredients?: StepIngredient[];
}

export interface RecipeDetail extends Recipe {
  steps: (RecipeStep & { ingredients: StepIngredient[] })[];
}

export interface StepIngredient {
  id: string;
  step_id: string;
  ingredient_id: string;
  unit_id?: string;
  amount?: number;
  notes?: string;
  ingredient_name?: string;
  unit_name?: string;
  unit_abbr?: string;
}

export interface Ingredient {
  id: string;
  name_en: string;
  name_de?: string;
  plural_en?: string;
  plural_de?: string;
  default_unit_id?: string;
  suggested_unit_id?: string;
}

export interface Unit {
  id: string;
  name_en: string;
  name_de?: string;
  abbreviation_en?: string;
  abbreviation_de?: string;
}

export interface InviteLink {
  id: string;
  token: string;
  max_uses: number;
  use_count: number;
  expires_at?: string;
  created_at: string;
}

export interface SystemSettings {
  open_registration: boolean;
  allow_public_recipes: boolean;
  mcp_enabled: boolean;
}
