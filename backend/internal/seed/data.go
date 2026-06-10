package seed

// seedIngredient is one line of a seed recipe (quantities are for Base persons).
type seedIngredient struct {
	Name string
	Qty  float64
	Unit string
}

type seedRecipe struct {
	Name        string
	Base        int
	Kind        string // plat | dessert | apero
	Instr       string
	Ingredients []seedIngredient
}

// canonicalUnits is the units referential (Excel "data" sheet).
var canonicalUnits = []string{
	"pièce", "g", "kg", "mL", "L", "cuillère à soupe", "cuillère à café",
	"tranche", "gousse", "botte", "sachet", "pincée", "verre", "barquette",
}

// canonicalIngredients seeds the normalized referential so the fuzzy matcher
// nudges users toward these names (e.g. "beurre salé" -> "beurre demi-sel").
var canonicalIngredients = []struct {
	Name string
	Unit string
}{
	{"beurre demi-sel", "g"},
	{"oignons blancs", "pièce"},
	{"lardons", "g"},
	{"allumette bacon", "g"},
	{"crème fraîche", "mL"},
	{"reblochon", "pièce"},
	{"pommes de terre", "kg"},
	{"vin blanc", "mL"},
	{"gruyère râpé", "g"},
}

// seedRecipes is the starter library extracted from the reference Excel files.
var seedRecipes = []seedRecipe{
	{
		Name: "Fondue vigneronne", Base: 6, Kind: "plat",
		Instr: "Faire chauffer le bouillon. Chacun cuit ses morceaux de viande à la fourchette, accompagnés de sauces.",
		Ingredients: []seedIngredient{
			{"viande à fondue (boeuf)", 250, "g"},
			{"bouillon de boeuf", 1500, "mL"},
			{"sauces variées", 6, "pièce"},
			{"pommes de terre", 1, "kg"},
		},
	},
	{
		Name: "Fondue Savoyarde", Base: 6, Kind: "plat",
		Instr: "Frotter le caquelon à l'ail, faire fondre les fromages avec le vin blanc, servir avec du pain.",
		Ingredients: []seedIngredient{
			{"comté", 200, "g"},
			{"beaufort", 200, "g"},
			{"emmental", 200, "g"},
			{"vin blanc", 400, "mL"},
			{"ail", 1, "gousse"},
			{"pain", 1, "kg"},
		},
	},
	{
		Name: "Croziflette", Base: 6, Kind: "plat",
		Instr: "Cuire les crozets, faire revenir lardons et oignons, mélanger avec la crème, recouvrir de reblochon, gratiner.",
		Ingredients: []seedIngredient{
			{"crozets", 500, "g"},
			{"reblochon", 1, "pièce"},
			{"lardons", 200, "g"},
			{"oignons blancs", 2, "pièce"},
			{"crème fraîche", 200, "mL"},
		},
	},
	{
		Name: "Soupe oignons", Base: 6, Kind: "plat",
		Instr: "Faire revenir les oignons émincés dans le beurre, mouiller au bouillon, servir gratiné au fromage.",
		Ingredients: []seedIngredient{
			{"oignons blancs", 6, "pièce"},
			{"beurre demi-sel", 50, "g"},
			{"bouillon de boeuf", 1500, "mL"},
			{"gruyère râpé", 150, "g"},
			{"pain", 6, "tranche"},
		},
	},
	{
		Name: "Raclette", Base: 6, Kind: "plat",
		Instr: "Faire fondre le fromage à raclette, servir avec pommes de terre et charcuterie.",
		Ingredients: []seedIngredient{
			{"fromage à raclette", 200, "g"},
			{"pommes de terre", 1.5, "kg"},
			{"charcuterie assortie", 150, "g"},
		},
	},
	{
		Name: "Galettes Bretonnes", Base: 6, Kind: "plat",
		Instr: "Préparer la pâte de sarrasin, garnir jambon-oeuf-fromage, plier et servir.",
		Ingredients: []seedIngredient{
			{"farine de sarrasin", 500, "g"},
			{"oeufs", 6, "pièce"},
			{"jambon", 6, "tranche"},
			{"gruyère râpé", 200, "g"},
			{"beurre demi-sel", 50, "g"},
		},
	},
	{
		Name: "Pâtes Bolognaise", Base: 6, Kind: "plat",
		Instr: "Faire revenir la viande hachée avec oignons, ajouter la sauce tomate, mijoter, servir sur les pâtes.",
		Ingredients: []seedIngredient{
			{"pâtes", 500, "g"},
			{"viande hachée", 500, "g"},
			{"sauce tomate", 800, "g"},
			{"oignons blancs", 2, "pièce"},
		},
	},
	{
		Name: "Pâtes Carbonara", Base: 6, Kind: "plat",
		Instr: "Cuire les pâtes, mélanger lardons, oeufs et parmesan hors du feu.",
		Ingredients: []seedIngredient{
			{"pâtes", 500, "g"},
			{"lardons", 200, "g"},
			{"oeufs", 4, "pièce"},
			{"parmesan", 100, "g"},
		},
	},
	{
		Name: "Tarte flambée", Base: 6, Kind: "plat",
		Instr: "Étaler la pâte fine, garnir crème, oignons et lardons, cuire à four très chaud.",
		Ingredients: []seedIngredient{
			{"pâte à tarte flambée", 2, "pièce"},
			{"crème fraîche", 200, "mL"},
			{"oignons blancs", 2, "pièce"},
			{"allumette bacon", 200, "g"},
		},
	},
	{
		Name: "Préfou/Préfoutre", Base: 6, Kind: "apero",
		Instr: "Garnir le pain de beurre à l'ail, enfourner jusqu'à ce qu'il soit doré.",
		Ingredients: []seedIngredient{
			{"pain préfou", 2, "pièce"},
			{"beurre demi-sel", 100, "g"},
			{"ail", 3, "gousse"},
		},
	},
	{
		Name: "Côte de Boeuf", Base: 6, Kind: "plat",
		Instr: "Saisir la côte de boeuf à feu vif, reposer, trancher, servir avec accompagnement.",
		Ingredients: []seedIngredient{
			{"côte de boeuf", 1.2, "kg"},
			{"beurre demi-sel", 50, "g"},
			{"pommes de terre", 1, "kg"},
		},
	},
	{
		Name: "FajiChilli", Base: 6, Kind: "plat",
		Instr: "Mélange fajitas/chili : faire revenir viande et poivrons, épices, haricots, servir en tortillas.",
		Ingredients: []seedIngredient{
			{"viande hachée", 500, "g"},
			{"poivrons", 3, "pièce"},
			{"haricots rouges", 400, "g"},
			{"tortillas", 12, "pièce"},
			{"oignons blancs", 2, "pièce"},
		},
	},
	{
		Name: "Tartiflette", Base: 6, Kind: "plat",
		Instr: "Cuire les pommes de terre, faire revenir lardons et oignons, recouvrir de reblochon, gratiner.",
		Ingredients: []seedIngredient{
			{"pommes de terre", 1.5, "kg"},
			{"reblochon", 1, "pièce"},
			{"lardons", 200, "g"},
			{"oignons blancs", 2, "pièce"},
			{"crème fraîche", 200, "mL"},
		},
	},
	{
		Name: "Connard Laquet (magret de canard laqué)", Base: 6, Kind: "plat",
		Instr: "Quadriller les magrets, cuire côté peau, laquer au miel et sauce soja, trancher.",
		Ingredients: []seedIngredient{
			{"magret de canard", 3, "pièce"},
			{"miel", 4, "cuillère à soupe"},
			{"sauce soja", 4, "cuillère à soupe"},
		},
	},
	{
		Name: "Croissant saumon apéro", Base: 6, Kind: "apero",
		Instr: "Garnir mini-croissants de saumon fumé et fromage frais.",
		Ingredients: []seedIngredient{
			{"mini-croissants", 18, "pièce"},
			{"saumon fumé", 150, "g"},
			{"fromage frais", 150, "g"},
		},
	},
	{
		Name: "Roulé wrap apéro", Base: 6, Kind: "apero",
		Instr: "Tartiner les wraps, garnir, rouler serré et trancher en bouchées.",
		Ingredients: []seedIngredient{
			{"tortillas", 4, "pièce"},
			{"fromage frais", 150, "g"},
			{"jambon", 4, "tranche"},
			{"salade", 0.5, "botte"},
		},
	},
	{
		Name: "Salade de fred (pâtes tortillis)", Base: 6, Kind: "plat",
		Instr: "Cuire les tortillis, mélanger avec crudités, dés de fromage et vinaigrette.",
		Ingredients: []seedIngredient{
			{"pâtes tortillis", 500, "g"},
			{"tomates", 4, "pièce"},
			{"maïs", 300, "g"},
			{"emmental", 150, "g"},
			{"vinaigrette", 100, "mL"},
		},
	},
	{
		Name: "Crêpes by Papy", Base: 6, Kind: "dessert",
		Instr: "Préparer la pâte, laisser reposer, cuire les crêpes une à une.",
		Ingredients: []seedIngredient{
			{"farine", 500, "g"},
			{"oeufs", 4, "pièce"},
			{"lait", 1, "L"},
			{"beurre demi-sel", 50, "g"},
			{"sucre", 100, "g"},
		},
	},
	{
		Name: "Galette des rois", Base: 8, Kind: "dessert",
		Instr: "Garnir deux pâtes feuilletées de frangipane, dorer à l'oeuf, cuire au four.",
		Ingredients: []seedIngredient{
			{"pâte feuilletée", 2, "pièce"},
			{"poudre d'amande", 200, "g"},
			{"beurre demi-sel", 100, "g"},
			{"oeufs", 3, "pièce"},
			{"sucre", 100, "g"},
		},
	},
	{
		Name: "Mousse au chocolat", Base: 6, Kind: "dessert",
		Instr: "Faire fondre le chocolat, incorporer les jaunes puis les blancs en neige, réfrigérer.",
		Ingredients: []seedIngredient{
			{"chocolat noir", 200, "g"},
			{"oeufs", 6, "pièce"},
			{"sucre", 50, "g"},
		},
	},
	// ── Cocktails ──────────────────────────────────────────────────────────
	{
		Name: "Mojito", Base: 1, Kind: "cocktail",
		Instr: "Piler menthe, citron vert et sucre.\nAjouter le rhum et la glace pilée.\nCompléter à l'eau gazeuse, mélanger.",
		Ingredients: []seedIngredient{
			{"rhum blanc", 50, "mL"},
			{"citron vert", 1, "pièce"},
			{"menthe fraîche", 6, "pièce"},
			{"sucre de canne", 2, "cuillère à café"},
			{"eau gazeuse", 100, "mL"},
		},
	},
	{
		Name: "Margarita", Base: 1, Kind: "cocktail",
		Instr: "Givrer le verre au sel.\nShaker tequila, triple sec et jus de citron vert avec de la glace.\nVerser dans le verre.",
		Ingredients: []seedIngredient{
			{"tequila", 50, "mL"},
			{"triple sec", 20, "mL"},
			{"citron vert", 1, "pièce"},
			{"sel", 1, "pincée"},
		},
	},
	{
		Name: "Spritz", Base: 1, Kind: "cocktail",
		Instr: "Remplir un verre de glace.\nVerser l'Apérol puis le prosecco.\nTrait d'eau gazeuse, tranche d'orange.",
		Ingredients: []seedIngredient{
			{"apérol", 60, "mL"},
			{"prosecco", 90, "mL"},
			{"eau gazeuse", 30, "mL"},
			{"orange", 1, "pièce"},
		},
	},
	{
		Name: "Gin Tonic", Base: 1, Kind: "cocktail",
		Instr: "Gin sur glace.\nCompléter au tonic.\nAjouter une rondelle de citron vert.",
		Ingredients: []seedIngredient{
			{"gin", 50, "mL"},
			{"tonic", 150, "mL"},
			{"citron vert", 1, "pièce"},
		},
	},
	{
		Name: "Vodka Redbull", Base: 1, Kind: "cocktail",
		Instr: "Vodka sur glace.\nCompléter au Red Bull.",
		Ingredients: []seedIngredient{
			{"vodka", 40, "mL"},
			{"red bull", 200, "mL"},
		},
	},
}

// recipePhotos maps a seed recipe to a topical internet image (loremflickr
// serves a real Flickr photo for the keyword; lock keeps it stable). Editable
// per recipe in the app — paste any image URL or upload a file.
var recipePhotos = map[string]string{
	"Fondue vigneronne":                          "https://loremflickr.com/640/480/fondue?lock=1",
	"Fondue Savoyarde":                           "https://loremflickr.com/640/480/fondue?lock=2",
	"Croziflette":                                "https://loremflickr.com/640/480/gratin?lock=3",
	"Soupe oignons":                              "https://loremflickr.com/640/480/soup?lock=4",
	"Raclette":                                   "https://loremflickr.com/640/480/raclette?lock=5",
	"Galettes Bretonnes":                         "https://loremflickr.com/640/480/crepe?lock=6",
	"Pâtes Bolognaise":                           "https://loremflickr.com/640/480/spaghetti?lock=7",
	"Pâtes Carbonara":                            "https://loremflickr.com/640/480/pasta?lock=8",
	"Tarte flambée":                              "https://loremflickr.com/640/480/flammkuchen?lock=9",
	"Préfou/Préfoutre":                           "https://loremflickr.com/640/480/garlicbread?lock=10",
	"Côte de Boeuf":                              "https://loremflickr.com/640/480/steak?lock=11",
	"FajiChilli":                                 "https://loremflickr.com/640/480/fajitas?lock=12",
	"Tartiflette":                                "https://loremflickr.com/640/480/tartiflette?lock=13",
	"Connard Laquet (magret de canard laqué)":    "https://loremflickr.com/640/480/duck?lock=14",
	"Croissant saumon apéro":                     "https://loremflickr.com/640/480/salmon?lock=15",
	"Roulé wrap apéro":                           "https://loremflickr.com/640/480/wrap?lock=16",
	"Salade de fred (pâtes tortillis)":           "https://loremflickr.com/640/480/pastasalad?lock=17",
	"Crêpes by Papy":                             "https://loremflickr.com/640/480/pancake?lock=18",
	"Galette des rois":                           "https://loremflickr.com/640/480/cake?lock=19",
	"Mousse au chocolat":                         "https://loremflickr.com/640/480/chocolate?lock=20",
	"Mojito":                                     "https://loremflickr.com/640/480/mojito?lock=21",
	"Margarita":                                  "https://loremflickr.com/640/480/margarita?lock=22",
	"Spritz":                                     "https://loremflickr.com/640/480/cocktail?lock=23",
	"Gin Tonic":                                  "https://loremflickr.com/640/480/gintonic?lock=24",
	"Vodka Redbull":                              "https://loremflickr.com/640/480/cocktail?lock=25",
}

// seedCatalogItem is one product of a reusable non-voted catalog.
type seedCatalogItem struct {
	Name     string
	Unit     string
	Section  string
	Quantity float64
}

type seedCatalog struct {
	Name     string
	Sections []string
	Items    []seedCatalogItem
}

// seedCatalogs are reusable, non-voted product lists (organizers set totals).
// "Indispensables" mirrors a "vie en communauté" sheet; adjust freely in-app.
var seedCatalogs = []seedCatalog{
	{
		Name:     "Apéro",
		Sections: []string{"Gâteaux apéro", "Bière", "Spiritueux", "Softs & sirops", "Fruits & extras", "Cocktail"},
		Items: []seedCatalogItem{
			// Gâteaux apéro
			{"chips", "paquet", "Gâteaux apéro", 4},
			{"cacahuètes", "paquet", "Gâteaux apéro", 2},
			{"gâteaux apéro", "paquet", "Gâteaux apéro", 3},
			{"saucisson", "pièce", "Gâteaux apéro", 2},
			// Bière
			{"guinness", "canette", "Bière", 12},
			{"bière", "fût 5L", "Bière", 8},
			// Spiritueux (sheet « Alcool et cocktails »)
			{"chartreuse", "bouteille", "Spiritueux", 0},
			{"genepi", "bouteille", "Spiritueux", 0},
			{"rhum blanc", "bouteille", "Spiritueux", 0},
			{"rhum ambré", "bouteille", "Spiritueux", 0},
			{"rhum spicy", "bouteille", "Spiritueux", 0},
			{"vodka", "bouteille", "Spiritueux", 1},
			{"manzana", "bouteille", "Spiritueux", 1},
			{"aperol/spritz", "bouteille", "Spiritueux", 0},
			{"prosecco", "bouteille", "Spiritueux", 0},
			{"cointreau", "bouteille", "Spiritueux", 0},
			{"vin rouge", "bouteille", "Spiritueux", 0},
			{"bailey's", "bouteille", "Spiritueux", 1},
			{"get27", "bouteille", "Spiritueux", 1},
			{"menthe pastille", "bouteille", "Spiritueux", 1},
			{"groscon", "bouteille", "Spiritueux", 1},
			// Softs & sirops
			{"sprite", "bouteille", "Softs & sirops", 1},
			{"coca pour les gros", "pack", "Softs & sirops", 1},
			{"orangina", "bouteille", "Softs & sirops", 0},
			{"sirop de grenadine", "bouteille", "Softs & sirops", 1},
			{"perrier fine bulle (c'est la bleue, connard)", "pack", "Softs & sirops", 2},
			{"oasis tropical de gros", "bouteille", "Softs & sirops", 2},
			{"ice tea pêche", "bouteille", "Softs & sirops", 2},
			{"SodaStream - si besoin", "machine", "Softs & sirops", 1},
			// Fruits & extras
			{"citrons verts", "pièce", "Fruits & extras", 10},
			{"citrons jaunes", "pièce", "Fruits & extras", 10},
			{"menthe", "sachet", "Fruits & extras", 2},
		},
	},
	{
		Name:     "Indispensables",
		Sections: []string{"Cuisine", "Hygiène"},
		Items: []seedCatalogItem{
			// « Vie en communoté »
			{"éponge", "unité", "Cuisine", 3},
			{"liquide vaisselle", "litre", "Cuisine", 1},
			{"sac poubelles (50L)", "rouleau", "Cuisine", 1},
			{"papier alu", "mètre", "Cuisine", 50},
			{"papier sulfurisé", "mètre", "Cuisine", 20},
			{"torchon", "unité", "Cuisine", 3},
			{"sopalin", "rouleau", "Cuisine", 6},
			{"pastilles lave-vaisselle", "paquet", "Cuisine", 1},
			{"filtre à café", "boîte", "Cuisine", 0},
			{"vinaigre balsamique", "bouteille", "Cuisine", 1},
			{"savon pour les mains", "savon", "Hygiène", 2},
			{"PQ", "rouleau", "Hygiène", 12},
			{"mouchoirs", "paquet", "Hygiène", 1},
			{"désodorisant toilette GRO-KK", "bouteille", "Hygiène", 2},
		},
	},
}
