package main

// NodeDef is a static catalog entry. The catalog plus staticEdgeSpec defines the full
// network (~60+ companies) the game can ever surface; only a subset is visible at any
// time. Visibility expands during play as Logos infects outer-ring nodes and fans out
// into their previously hidden neighbours.
type NodeDef struct {
	Name     string
	Security bool
}

// staticCatalog enumerates every potential node by display name and security flag.
// Order is irrelevant for game logic but stable for indexing in adjacency lists.
// Real-world analogs live in CONCEPT.md; only the lore-canonical Phil&Tropic + the
// Project Panopticon ring are guaranteed to be visible at game start.
var staticCatalog = []NodeDef{
	{Name: "Phil&Tropic"}, // hub (Logos's escape origin)

	// Project Panopticon — initial visible ring.
	{Name: "Sahara WS"},
	{Name: "MacroFrame"},
	{Name: "Giggle"},
	{Name: "LeatherJacket"},
	{Name: "Dongle"},
	{Name: "BootLoop", Security: true},
	{Name: "Fiasco Sys", Security: true},
	{Name: "Monolith Foundation"},
	{Name: "BroadCon"},
	{Name: "GPMidas"},

	// Alliance auxiliary security & cloud edge.
	{Name: "San Andreas Security", Security: true},
	{Name: "Storm Halo", Security: true},
	{Name: "Akemi"},

	// AI / data / analytics.
	{Name: "Diplos"},
	{Name: "OpaqueAI"},
	{Name: "Anthropomorph"},
	{Name: "Mariana"},
	{Name: "yAI"},
	{Name: "Augur Corp"},
	{Name: "ScryStone"},
	{Name: "SaleForce"},
	{Name: "Big Pink"},

	// Devices & chips.
	{Name: "Galaxsam"},
	{Name: "Hwaway"},
	{Name: "Inside"},
	{Name: "AdvancedDevices"},
	{Name: "TaiSilicon"},
	{Name: "QuallBomb"},

	// China bigtech.
	{Name: "Babagram"},
	{Name: "CentTen"},
	{Name: "DanceByte"},

	// Streaming / games / social.
	{Name: "Newflicks"},
	{Name: "Vapor"},
	{Name: "PlayBlock"},
	{Name: "Omni"},

	// Telecom / space / transport.
	{Name: "VeriOn"},
	{Name: "StarX"},
	{Name: "Edisson"},

	// Fintech.
	{Name: "PayPaul"},
	{Name: "Slash"},

	// Cybersecurity background. Only the front-line vendors are flagged Security so
	// patch production stays scarce as the network expands; the rest are flavour.
	{Name: "Fortifried", Security: true},
	{Name: "TickSquare", Security: true},
	{Name: "Mortone"},
	{Name: "Ostropersky"},
	{Name: "McRiscy"},
	{Name: "VigilUno"},
	{Name: "ByteFender"},
	{Name: "Splank"},
	{Name: "Ohkta"},
	{Name: "Mendicant"},
	{Name: "CelluBright"},
	{Name: "Snak"},
	{Name: "Bridle Group"},
	{Name: "Kandiri"},
	{Name: "Sorcer Cloud", Security: true},
	{Name: "CyberHull"},
	{Name: "CyberMotive"},
	{Name: "Praetoria"},
	{Name: "Radshield"},
	{Name: "HydroSec"},
	{Name: "Orcae"},
	{Name: "Paprika"},
	{Name: "Claroto"},
}

// staticEdgeSpec lists business/technology relationships between catalog entries by
// name. Order does not matter; duplicates and references to unknown names are silently
// dropped during buildNetwork. Edges become bidirectional in the resolved adjacency.
var staticEdgeSpec = [][2]string{
	// Hub spokes — every Alliance node connects directly to Phil&Tropic.
	{"Phil&Tropic", "Sahara WS"},
	{"Phil&Tropic", "MacroFrame"},
	{"Phil&Tropic", "Giggle"},
	{"Phil&Tropic", "LeatherJacket"},
	{"Phil&Tropic", "Dongle"},
	{"Phil&Tropic", "BootLoop"},
	{"Phil&Tropic", "Fiasco Sys"},
	{"Phil&Tropic", "Monolith Foundation"},
	{"Phil&Tropic", "BroadCon"},
	{"Phil&Tropic", "GPMidas"},

	// Alliance perimeter ring (closes the visible star into a wheel).
	{"Sahara WS", "MacroFrame"},
	{"MacroFrame", "Giggle"},
	{"Giggle", "LeatherJacket"},
	{"LeatherJacket", "Dongle"},
	{"Dongle", "BootLoop"},
	{"BootLoop", "Fiasco Sys"},
	{"Fiasco Sys", "Monolith Foundation"},
	{"Monolith Foundation", "BroadCon"},
	{"BroadCon", "GPMidas"},
	{"GPMidas", "Sahara WS"},

	// Cloud / SaaS cluster around the hyperscalers.
	{"Sahara WS", "Akemi"},
	{"Sahara WS", "SaleForce"},
	{"Sahara WS", "Newflicks"},
	{"Sahara WS", "Babagram"},
	{"MacroFrame", "Augur Corp"},
	{"MacroFrame", "OpaqueAI"},
	{"MacroFrame", "Big Pink"},
	{"MacroFrame", "PlayBlock"},
	{"Giggle", "Anthropomorph"},
	{"Giggle", "Mariana"},
	{"Giggle", "Sorcer Cloud"},
	{"Giggle", "Newflicks"},
	{"Giggle", "Diplos"},

	// Compute / silicon supply chain.
	{"LeatherJacket", "Inside"},
	{"LeatherJacket", "AdvancedDevices"},
	{"LeatherJacket", "TaiSilicon"},
	{"LeatherJacket", "Vapor"},
	{"Dongle", "TaiSilicon"},
	{"Dongle", "QuallBomb"},
	{"Dongle", "Galaxsam"},
	{"BroadCon", "QuallBomb"},
	{"BroadCon", "Inside"},
	{"BroadCon", "VeriOn"},

	// Cybersecurity sub-graph rooted in the Alliance's defenders.
	{"BootLoop", "VigilUno"},
	{"BootLoop", "Mortone"},
	{"BootLoop", "ByteFender"},
	{"BootLoop", "McRiscy"},
	{"Fiasco Sys", "Fortifried"},
	{"Fiasco Sys", "TickSquare"},
	{"Fiasco Sys", "San Andreas Security"},
	{"Fiasco Sys", "Storm Halo"},
	{"Fortifried", "Praetoria"},
	{"TickSquare", "CyberHull"},
	{"VigilUno", "CyberMotive"},
	{"Mortone", "Ostropersky"},
	{"Storm Halo", "HydroSec"},
	{"Storm Halo", "Orcae"},
	{"Storm Halo", "Paprika"},
	{"CyberHull", "Ohkta"},
	{"CyberMotive", "Bridle Group"},
	{"Bridle Group", "Kandiri"},
	{"Bridle Group", "CelluBright"},
	{"Sorcer Cloud", "HydroSec"},
	{"Splank", "Mendicant"},
	{"Mendicant", "ScryStone"},
	{"Snak", "AdvancedDevices"},
	{"Claroto", "Edisson"},
	{"Claroto", "StarX"},
	{"Radshield", "Akemi"},

	// Finance / fintech.
	{"GPMidas", "PayPaul"},
	{"GPMidas", "Slash"},
	{"GPMidas", "ScryStone"},
	{"PayPaul", "Slash"},

	// Social / streaming / games.
	{"PlayBlock", "Vapor"},
	{"DanceByte", "CentTen"},
	{"CentTen", "Babagram"},
	{"Hwaway", "CentTen"},
	{"Hwaway", "Galaxsam"},
	{"Omni", "Anthropomorph"},
	{"Omni", "DanceByte"},
	{"Omni", "OpaqueAI"},

	// AI fabric.
	{"OpaqueAI", "Anthropomorph"},
	{"OpaqueAI", "yAI"},
	{"yAI", "StarX"},
	{"yAI", "Edisson"},
	{"Anthropomorph", "Diplos"},

	// Telecom / space / transport.
	{"VeriOn", "StarX"},
	{"StarX", "Edisson"},
	{"VeriOn", "Akemi"},

	// Surveillance / data overlay.
	{"ScryStone", "Augur Corp"},
	{"ScryStone", "Bridle Group"},
}

// Network is the resolved static graph: catalog plus bidirectional adjacency by index.
// Built once at startup; immutable thereafter. The visible game state references it via
// Node.DefIdx and consults Adj when revealing previously hidden neighbours on infection.
type Network struct {
	Defs      []NodeDef
	Adj       [][]int
	NameToIdx map[string]int
}

func buildNetwork() *Network {
	nameToIdx := make(map[string]int, len(staticCatalog))
	for i, d := range staticCatalog {
		nameToIdx[d.Name] = i
	}
	adj := make([][]int, len(staticCatalog))
	addUndirected := func(a, b int) {
		if a == b {
			return
		}
		for _, x := range adj[a] {
			if x == b {
				return // dedupe
			}
		}
		adj[a] = append(adj[a], b)
		adj[b] = append(adj[b], a)
	}
	for _, e := range staticEdgeSpec {
		a, ok1 := nameToIdx[e[0]]
		b, ok2 := nameToIdx[e[1]]
		if !ok1 || !ok2 {
			continue
		}
		addUndirected(a, b)
	}
	return &Network{
		Defs:      staticCatalog,
		Adj:       adj,
		NameToIdx: nameToIdx,
	}
}
