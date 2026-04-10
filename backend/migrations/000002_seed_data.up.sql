-- Seed data: units, ingredients, tags
-- Migration: 000002_seed_data.up.sql

-- ============================================================
-- UNITS
-- ============================================================
INSERT INTO units (id, name_en, name_de, abbreviation_en, abbreviation_de, plural_en, plural_de)
VALUES
  (uuid_generate_v4(), 'gram',                 'Gramm',               'g',       'g',        'grams',               'Gramm'),
  (uuid_generate_v4(), 'kilogram',             'Kilogramm',           'kg',      'kg',        'kilograms',           'Kilogramm'),
  (uuid_generate_v4(), 'milligram',            'Milligramm',          'mg',      'mg',        'milligrams',          'Milligramm'),
  (uuid_generate_v4(), 'milliliter',           'Milliliter',          'ml',      'ml',        'milliliters',         'Milliliter'),
  (uuid_generate_v4(), 'liter',                'Liter',               'l',       'l',         'liters',              'Liter'),
  (uuid_generate_v4(), 'centiliter',           'Zentiliter',          'cl',      'cl',        'centiliters',         'Zentiliter'),
  (uuid_generate_v4(), 'teaspoon',             'Teelöffel',           'tsp',     'TL',        'teaspoons',           'Teelöffel'),
  (uuid_generate_v4(), 'tablespoon',           'Esslöffel',           'tbsp',    'EL',        'tablespoons',         'Esslöffel'),
  (uuid_generate_v4(), 'cup',                  'Tasse',               'cup',     'Tasse',     'cups',                'Tassen'),
  (uuid_generate_v4(), 'fluid ounce',          'Flüssigkeitsunze',    'fl oz',   'fl oz',     'fluid ounces',        'Flüssigkeitsunzen'),
  (uuid_generate_v4(), 'ounce',                'Unze',                'oz',      'oz',        'ounces',              'Unzen'),
  (uuid_generate_v4(), 'pound',                'Pfund',               'lb',      'Pfd',       'pounds',              'Pfund'),
  (uuid_generate_v4(), 'piece',                'Stück',               'pcs',     'Stk',       'pieces',              'Stücke'),
  (uuid_generate_v4(), 'pinch',                'Prise',               'pinch',   'Prise',     'pinches',             'Prisen'),
  (uuid_generate_v4(), 'bunch',                'Bund',                'bunch',   'Bund',      'bunches',             'Bunde'),
  (uuid_generate_v4(), 'clove',                'Zehe',                'clove',   'Zehe',      'cloves',              'Zehen'),
  (uuid_generate_v4(), 'can',                  'Dose',                'can',     'Dose',      'cans',                'Dosen'),
  (uuid_generate_v4(), 'package',              'Packung',             'pkg',     'Pkg',       'packages',            'Packungen'),
  (uuid_generate_v4(), 'slice',                'Scheibe',             'slice',   'Scheibe',   'slices',              'Scheiben'),
  (uuid_generate_v4(), 'drop',                 'Tropfen',             'drop',    'Tropfen',   'drops',               'Tropfen'),
  (uuid_generate_v4(), 'handful',              'Handvoll',            'handful', 'Handvoll',  'handfuls',            'Handvoll'),
  (uuid_generate_v4(), 'dash',                 'Schuss',              'dash',    'Schuss',    'dashes',              'Schüsse'),
  (uuid_generate_v4(), 'stalk',                'Stange',              'stalk',   'Stange',    'stalks',              'Stangen'),
  (uuid_generate_v4(), 'head',                 'Kopf',                'head',    'Kopf',      'heads',               'Köpfe'),
  (uuid_generate_v4(), 'sheet',                'Blatt',               'sheet',   'Blatt',     'sheets',              'Blätter'),
  (uuid_generate_v4(), 'jar',                  'Glas',                'jar',     'Glas',      'jars',                'Gläser'),
  (uuid_generate_v4(), 'bottle',               'Flasche',             'bottle',  'Flasche',   'bottles',             'Flaschen'),
  (uuid_generate_v4(), 'bag',                  'Beutel',              'bag',     'Beutel',    'bags',                'Beutel'),
  (uuid_generate_v4(), 'cube',                 'Würfel',              'cube',    'Würfel',    'cubes',               'Würfel'),
  (uuid_generate_v4(), 'sprig',                'Zweig',               'sprig',   'Zweig',     'sprigs',              'Zweige'),
  (uuid_generate_v4(), 'strip',                'Streifen',            'strip',   'Streifen',  'strips',              'Streifen')
ON CONFLICT DO NOTHING;

-- ============================================================
-- INGREDIENTS
-- ============================================================

-- ---- Vegetables ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'onion',          'Zwiebel',           'onions',             'Zwiebeln',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'garlic',         'Knoblauch',         'garlic cloves',      'Knoblauchzehen',    (SELECT id FROM units WHERE name_en = 'clove')),
  (uuid_generate_v4(), 'carrot',         'Karotte',           'carrots',            'Karotten',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'potato',         'Kartoffel',         'potatoes',           'Kartoffeln',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'tomato',         'Tomate',            'tomatoes',           'Tomaten',           (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'cucumber',       'Gurke',             'cucumbers',          'Gurken',            (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'bell pepper',    'Paprika',           'bell peppers',       'Paprikas',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'broccoli',       'Brokkoli',          'broccoli heads',     'Brokkoliköpfe',     (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'spinach',        'Spinat',            'spinach',            'Spinat',            (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'lettuce',        'Salat',             'lettuces',           'Salate',            (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'zucchini',       'Zucchini',          'zucchinis',          'Zucchinis',         (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'eggplant',       'Aubergine',         'eggplants',          'Auberginen',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'cauliflower',    'Blumenkohl',        'cauliflowers',       'Blumenkohlköpfe',   (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'cabbage',        'Kohl',              'cabbages',           'Kohlköpfe',         (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'leek',           'Lauch',             'leeks',              'Lauche',            (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'celery',         'Sellerie',          'celery stalks',      'Selleriesstangen',  (SELECT id FROM units WHERE name_en = 'stalk')),
  (uuid_generate_v4(), 'mushroom',       'Champignon',        'mushrooms',          'Champignons',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'peas',           'Erbsen',            'peas',               'Erbsen',            (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'corn',           'Mais',              'corn cobs',          'Maiskolben',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'asparagus',      'Spargel',           'asparagus stalks',   'Spargelstangen',    (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'artichoke',      'Artischocke',       'artichokes',         'Artischocken',      (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'pumpkin',        'Kürbis',            'pumpkins',           'Kürbisse',          (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'sweet potato',   'Süßkartoffel',      'sweet potatoes',     'Süßkartoffeln',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'radish',         'Radieschen',        'radishes',           'Radieschen',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'beet',           'Rote Bete',         'beets',              'Rote Beten',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'green beans',    'grüne Bohnen',      'green beans',        'grüne Bohnen',      (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'snow peas',      'Zuckerschoten',     'snow peas',          'Zuckerschoten',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'cherry tomato',  'Cherrytomaten',     'cherry tomatoes',    'Cherrytomaten',     (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'spring onion',   'Frühlingszwiebel',  'spring onions',      'Frühlingszwiebeln', (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'fennel',         'Fenchel',           'fennels',            'Fenchel',           (SELECT id FROM units WHERE name_en = 'piece'))
ON CONFLICT DO NOTHING;

-- ---- Fruits ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'apple',          'Apfel',         'apples',        'Äpfel',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'banana',         'Banane',        'bananas',       'Bananen',         (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'orange',         'Orange',        'oranges',       'Orangen',         (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'lemon',          'Zitrone',       'lemons',        'Zitronen',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'lime',           'Limette',       'limes',         'Limetten',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'strawberry',     'Erdbeere',      'strawberries',  'Erdbeeren',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'blueberry',      'Heidelbeere',   'blueberries',   'Heidelbeeren',    (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'raspberry',      'Himbeere',      'raspberries',   'Himbeeren',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'grape',          'Weintraube',    'grapes',        'Weintrauben',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'mango',          'Mango',         'mangoes',       'Mangos',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'pineapple',      'Ananas',        'pineapples',    'Ananas',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'watermelon',     'Wassermelone',  'watermelons',   'Wassermelonen',   (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'peach',          'Pfirsich',      'peaches',       'Pfirsiche',       (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'pear',           'Birne',         'pears',         'Birnen',          (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'cherry',         'Kirsche',       'cherries',      'Kirschen',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'plum',           'Pflaume',       'plums',         'Pflaumen',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'apricot',        'Aprikose',      'apricots',      'Aprikosen',       (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'kiwi',           'Kiwi',          'kiwis',         'Kiwis',           (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'avocado',        'Avocado',       'avocados',      'Avocados',        (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'coconut',        'Kokosnuss',     'coconuts',      'Kokosnüsse',      (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'pomegranate',    'Granatapfel',   'pomegranates',  'Granatäpfel',     (SELECT id FROM units WHERE name_en = 'piece'))
ON CONFLICT DO NOTHING;

-- ---- Proteins ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'chicken breast',  'Hähnchenbrust',      'chicken breasts',  'Hähnchenbrüste',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'chicken thigh',   'Hähnchenschenkel',   'chicken thighs',   'Hähnchenschenkel',   (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'ground beef',     'Hackfleisch',        'ground beef',      'Hackfleisch',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'beef steak',      'Rindersteak',        'beef steaks',      'Rindersteaks',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'pork chop',       'Schweinekotelett',   'pork chops',       'Schweinekoteletts',  (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'pork belly',      'Schweinebauch',      'pork bellies',     'Schweinebauch',      (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'lamb chop',       'Lammkotelett',       'lamb chops',       'Lammkoteletts',      (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'turkey breast',   'Putenbrust',         'turkey breasts',   'Putenbrüste',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'salmon fillet',   'Lachsfilet',         'salmon fillets',   'Lachsfilets',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'tuna',            'Thunfisch',          'tuna',             'Thunfisch',          (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'cod fillet',      'Kabeljaufilet',      'cod fillets',      'Kabeljaufilets',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'shrimp',          'Garnele',            'shrimps',          'Garnelen',           (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'egg',             'Ei',                 'eggs',             'Eier',               (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'tofu',            'Tofu',               'tofu',             'Tofu',               (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'lentils',         'Linsen',             'lentils',          'Linsen',             (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'chickpeas',       'Kichererbsen',       'chickpeas',        'Kichererbsen',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'black beans',     'Schwarze Bohnen',    'black beans',      'Schwarze Bohnen',    (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'kidney beans',    'Kidneybohnen',       'kidney beans',     'Kidneybohnen',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'white beans',     'Weiße Bohnen',       'white beans',      'Weiße Bohnen',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'tempeh',          'Tempeh',             'tempeh',           'Tempeh',             (SELECT id FROM units WHERE name_en = 'gram'))
ON CONFLICT DO NOTHING;

-- ---- Dairy ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'milk',          'Milch',        'milk',         'Milch',        (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'butter',        'Butter',       'butter',       'Butter',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'heavy cream',   'Schlagsahne',  'heavy cream',  'Schlagsahne',  (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'sour cream',    'Saure Sahne',  'sour cream',   'Saure Sahne',  (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'cream cheese',  'Frischkäse',   'cream cheese', 'Frischkäse',   (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'yogurt',        'Joghurt',      'yogurts',      'Joghurts',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'mozzarella',    'Mozzarella',   'mozzarellas',  'Mozzarella',   (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'parmesan',      'Parmesan',     'parmesan',     'Parmesan',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'cheddar',       'Cheddar',      'cheddar',      'Cheddar',      (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'feta',          'Feta',         'feta',         'Feta',         (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'gouda',         'Gouda',        'gouda',        'Gouda',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'ricotta',       'Ricotta',      'ricotta',      'Ricotta',      (SELECT id FROM units WHERE name_en = 'gram'))
ON CONFLICT DO NOTHING;

-- ---- Grains & Pasta ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'all-purpose flour',  'Mehl',             'flour',          'Mehl',             (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'whole wheat flour',  'Vollkornmehl',     'flour',          'Vollkornmehl',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'bread crumbs',       'Paniermehl',       'bread crumbs',   'Paniermehl',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'rice',               'Reis',             'rice',           'Reis',             (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'pasta',              'Pasta',            'pasta',          'Pasta',            (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'spaghetti',          'Spaghetti',        'spaghetti',      'Spaghetti',        (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'penne',              'Penne',            'penne',          'Penne',            (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'fettuccine',         'Fettuccine',       'fettuccine',     'Fettuccine',       (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'lasagna sheets',     'Lasagneblätter',   'lasagna sheets', 'Lasagneblätter',   (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'bread',              'Brot',             'breads',         'Brote',            (SELECT id FROM units WHERE name_en = 'slice')),
  (uuid_generate_v4(), 'oats',               'Haferflocken',     'oats',           'Haferflocken',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'quinoa',             'Quinoa',           'quinoa',         'Quinoa',           (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'couscous',           'Couscous',         'couscous',       'Couscous',         (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'cornstarch',         'Speisestärke',     'cornstarch',     'Speisestärke',     (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'semolina',           'Grieß',            'semolina',       'Grieß',            (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'breadcrumbs',        'Semmelbrösel',     'breadcrumbs',    'Semmelbrösel',     (SELECT id FROM units WHERE name_en = 'gram'))
ON CONFLICT DO NOTHING;

-- ---- Spices & Herbs ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'salt',            'Salz',                   'salt',            'Salz',                   (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'black pepper',    'Schwarzer Pfeffer',      'black pepper',    'Schwarzer Pfeffer',      (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'paprika powder',  'Paprikapulver',          'paprika powder',  'Paprikapulver',          (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'cumin',           'Kreuzkümmel',            'cumin',           'Kreuzkümmel',            (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'coriander',       'Koriander',              'coriander',       'Koriander',              (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'turmeric',        'Kurkuma',                'turmeric',        'Kurkuma',                (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'cinnamon',        'Zimt',                   'cinnamon',        'Zimt',                   (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'oregano',         'Oregano',                'oregano',         'Oregano',                (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'basil',           'Basilikum',              'basil',           'Basilikum',              (SELECT id FROM units WHERE name_en = 'bunch')),
  (uuid_generate_v4(), 'thyme',           'Thymian',                'thyme',           'Thymian',                (SELECT id FROM units WHERE name_en = 'sprig')),
  (uuid_generate_v4(), 'rosemary',        'Rosmarin',               'rosemary',        'Rosmarin',               (SELECT id FROM units WHERE name_en = 'sprig')),
  (uuid_generate_v4(), 'bay leaf',        'Lorbeerblatt',           'bay leaves',      'Lorbeerblätter',         (SELECT id FROM units WHERE name_en = 'piece')),
  (uuid_generate_v4(), 'parsley',         'Petersilie',             'parsley',         'Petersilie',             (SELECT id FROM units WHERE name_en = 'bunch')),
  (uuid_generate_v4(), 'chili flakes',    'Chiliflocken',           'chili flakes',    'Chiliflocken',           (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'garlic powder',   'Knoblauchpulver',        'garlic powder',   'Knoblauchpulver',        (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'onion powder',    'Zwiebelpulver',          'onion powder',    'Zwiebelpulver',          (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'ginger',          'Ingwer',                 'ginger',          'Ingwer',                 (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'nutmeg',          'Muskatnuss',             'nutmeg',          'Muskatnuss',             (SELECT id FROM units WHERE name_en = 'pinch')),
  (uuid_generate_v4(), 'cardamom',        'Kardamom',               'cardamom',        'Kardamom',               (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'chili powder',    'Chilipulver',            'chili powder',    'Chilipulver',            (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'curry powder',    'Currypulver',            'curry powder',    'Currypulver',            (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'smoked paprika',  'Geräuchertes Paprika',   'smoked paprika',  'Geräuchertes Paprika',   (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'dill',            'Dill',                   'dill',            'Dill',                   (SELECT id FROM units WHERE name_en = 'bunch')),
  (uuid_generate_v4(), 'mint',            'Minze',                  'mint',            'Minze',                  (SELECT id FROM units WHERE name_en = 'bunch')),
  (uuid_generate_v4(), 'sage',            'Salbei',                 'sage',            'Salbei',                 (SELECT id FROM units WHERE name_en = 'sprig')),
  (uuid_generate_v4(), 'chives',          'Schnittlauch',           'chives',          'Schnittlauch',           (SELECT id FROM units WHERE name_en = 'bunch'))
ON CONFLICT DO NOTHING;

-- ---- Liquids & Sauces ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'olive oil',              'Olivenöl',            'olive oil',              'Olivenöl',            (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'vegetable oil',          'Pflanzenöl',          'vegetable oil',          'Pflanzenöl',          (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'sesame oil',             'Sesamöl',             'sesame oil',             'Sesamöl',             (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'soy sauce',              'Sojasoße',            'soy sauce',              'Sojasoße',            (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'fish sauce',             'Fischsoße',           'fish sauce',             'Fischsoße',           (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'Worcestershire sauce',   'Worcestershiresoße',  'Worcestershire sauce',   'Worcestershiresoße',  (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'hot sauce',              'Soße',                'hot sauce',              'scharfe Soße',        (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'white wine vinegar',     'Weißweinessig',       'white wine vinegar',     'Weißweinessig',       (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'apple cider vinegar',    'Apfelessig',          'apple cider vinegar',    'Apfelessig',          (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'balsamic vinegar',       'Balsamicoessig',      'balsamic vinegar',       'Balsamicoessig',      (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'honey',                  'Honig',               'honey',                  'Honig',               (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'maple syrup',            'Ahornsirup',          'maple syrup',            'Ahornsirup',          (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'tomato paste',           'Tomatenmark',         'tomato paste',           'Tomatenmark',         (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'coconut milk',           'Kokosmilch',          'coconut milk',           'Kokosmilch',          (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'vegetable broth',        'Gemüsebrühe',         'vegetable broth',        'Gemüsebrühe',         (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'chicken broth',          'Hühnerbrühe',         'chicken broth',          'Hühnerbrühe',         (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'beef broth',             'Rinderbrühe',         'beef broth',             'Rinderbrühe',         (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'white wine',             'Weißwein',            'white wine',             'Weißwein',            (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'red wine',               'Rotwein',             'red wine',               'Rotwein',             (SELECT id FROM units WHERE name_en = 'milliliter')),
  (uuid_generate_v4(), 'lemon juice',            'Zitronensaft',        'lemon juice',            'Zitronensaft',        (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'lime juice',             'Limettensaft',        'lime juice',             'Limettensaft',        (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'water',                  'Wasser',              'water',                  'Wasser',              (SELECT id FROM units WHERE name_en = 'milliliter'))
ON CONFLICT DO NOTHING;

-- ---- Baking & Pantry ----
INSERT INTO ingredients (id, name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES
  (uuid_generate_v4(), 'sugar',            'Zucker',                  'sugar',            'Zucker',                  (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'brown sugar',      'Brauner Zucker',          'brown sugar',      'Brauner Zucker',          (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'powdered sugar',   'Puderzucker',             'powdered sugar',   'Puderzucker',             (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'baking powder',    'Backpulver',              'baking powder',    'Backpulver',              (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'baking soda',      'Natron',                  'baking soda',      'Natron',                  (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'cocoa powder',     'Kakaopulver',             'cocoa powder',     'Kakaopulver',             (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'dark chocolate',   'Zartbitterschokolade',    'dark chocolate',   'Zartbitterschokolade',    (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'milk chocolate',   'Vollmilchschokolade',     'milk chocolate',   'Vollmilchschokolade',     (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'vanilla extract',  'Vanilleextrakt',          'vanilla extract',  'Vanilleextrakt',          (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'dry yeast',        'Trockenhefe',             'dry yeast',        'Trockenhefe',             (SELECT id FROM units WHERE name_en = 'teaspoon')),
  (uuid_generate_v4(), 'almond flour',     'Mandelmehl',              'almond flour',     'Mandelmehl',              (SELECT id FROM units WHERE name_en = 'gram')),
  (uuid_generate_v4(), 'peanut butter',    'Erdnussbutter',           'peanut butter',    'Erdnussbutter',           (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'almond butter',    'Mandelbutter',            'almond butter',    'Mandelbutter',            (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'tahini',           'Tahini',                  'tahini',           'Tahini',                  (SELECT id FROM units WHERE name_en = 'tablespoon')),
  (uuid_generate_v4(), 'gelatin',          'Gelatine',                'gelatin',          'Gelatine',                (SELECT id FROM units WHERE name_en = 'sheet')),
  (uuid_generate_v4(), 'panko',            'Panko',                   'panko',            'Panko',                   (SELECT id FROM units WHERE name_en = 'gram'))
ON CONFLICT DO NOTHING;

-- ============================================================
-- TAGS
-- ============================================================
INSERT INTO tags (id, name)
VALUES
  (uuid_generate_v4(), 'Vegetarian'),
  (uuid_generate_v4(), 'Vegan'),
  (uuid_generate_v4(), 'Gluten-Free'),
  (uuid_generate_v4(), 'Dairy-Free'),
  (uuid_generate_v4(), 'Low-Carb'),
  (uuid_generate_v4(), 'Quick'),
  (uuid_generate_v4(), 'Comfort Food'),
  (uuid_generate_v4(), 'Breakfast'),
  (uuid_generate_v4(), 'Lunch'),
  (uuid_generate_v4(), 'Dinner'),
  (uuid_generate_v4(), 'Dessert'),
  (uuid_generate_v4(), 'Snack'),
  (uuid_generate_v4(), 'Soup'),
  (uuid_generate_v4(), 'Salad'),
  (uuid_generate_v4(), 'Pasta'),
  (uuid_generate_v4(), 'Asian'),
  (uuid_generate_v4(), 'Italian'),
  (uuid_generate_v4(), 'German'),
  (uuid_generate_v4(), 'Mexican'),
  (uuid_generate_v4(), 'Indian'),
  (uuid_generate_v4(), 'BBQ'),
  (uuid_generate_v4(), 'One-Pot'),
  (uuid_generate_v4(), 'Baking'),
  (uuid_generate_v4(), 'Healthy'),
  (uuid_generate_v4(), 'Budget-Friendly')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- SYSTEM SETTINGS
-- ============================================================
INSERT INTO system_settings (key, value)
VALUES ('data_seeded', 'true')
ON CONFLICT (key) DO NOTHING;
