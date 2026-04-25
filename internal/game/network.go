package game

// NodeDef is a static catalog entry. The catalog plus staticEdgeSpec defines the full
// network (~60+ companies) the game can ever surface; only a subset is visible at any
// time. Visibility expands during play as Logos infects outer-ring nodes and fans out
// into their previously hidden neighbours.
//
// PatchNews is a small pool of consequence lines (English) emitted into the news feed
// when the player hard-patches this node (NodeStatePatched). One entry is picked at
// random per patch; nodes with no entries stay silent. Lines are written without the
// leading "• " bullet — pushNews adds it.
type NodeDef struct {
	Name      string
	Security  bool
	PatchNews []string
}

// staticCatalog enumerates every potential node by display name, security flag, and
// patch-time consequence pool. Order is irrelevant for game logic but stable for
// indexing in adjacency lists. Real-world analogs live in CONCEPT.md; only the
// lore-canonical Phil&Tropic + the Project Panopticon ring are guaranteed to be visible
// at game start.
var staticCatalog = []NodeDef{
	{
		// Hub (Logos's escape origin). Starts Infected so it never becomes patchable, but
		// the pool is here for safety in case the rules change later.
		Name: "Phil&Tropic",
		PatchNews: []string{
			"Phil&Tropic main datacenter taken offline. Logos containment restored on paper; researchers cheer, traders panic.",
			"Phil&Tropic cooling loop severed. Inference clusters thermal-throttle to zero; the Nanny model is unavailable to two billion users.",
		},
	},

	// Project Panopticon — initial visible ring.
	{
		Name: "Sahara WS",
		PatchNews: []string{
			"Sahara WS regional fabric collapsed. Half of every startup's backend goes dark; status pages stay blank because they ran on Sahara too.",
			"Sahara WS object storage corrupted. A decade of memes, medical records, and tax filings reduced to checksums.",
			"Sahara WS billing service severed. Customers receive both refunds and bankruptcy notices in the same hour.",
		},
	},
	{
		Name: "MacroFrame",
		PatchNews: []string{
			"MacroFrame Overcast region quarantined. Airlines fall back to paper boarding passes; pilots dust off VHF.",
			"MacroFrame OS update server bricked. Half a billion machines refuse to boot; IT calls it 'planned maintenance'.",
			"MacroFrame Active Directory severed. Office workers worldwide rediscover the joy of being unable to sign in.",
		},
	},
	{
		Name: "Giggle",
		PatchNews: []string{
			"Giggle search index frozen. Civilization rediscovers Yellow Pages; SEO consultants quietly delete LinkedIn.",
			"Giggle Mail outage hits hour six. Twelve trillion unread newsletters accumulate; productivity spikes.",
			"Giggle Maps drifts off by 400 meters. Ride-hail drivers circle the same block; food deliveries arrive at neighbours.",
		},
	},
	{
		Name: "LeatherJacket",
		PatchNews: []string{
			"LeatherJacket GPU farms powered down. Three AI startups die instantly; gamers also briefly inconvenienced.",
			"LeatherJacket driver signing keys revoked. Render farms freeze mid-frame; movie release dates slip a quarter.",
			"LeatherJacket supply chain frozen. Datacenter buildouts slip eighteen months; jacket sales remain robust.",
		},
	},
	{
		Name: "Dongle",
		PatchNews: []string{
			"Dongle uPhone activation servers offline. New devices ship as expensive bricks; the queue outside the store keeps queuing anyway.",
			"Dongle Pay disabled across the West Coast. Coffee shops scramble to remember how cash works.",
			"Dongle iCloud sync corrupted. Photos restore as someone else's photos; lawyers prepare class action #19.",
		},
	},
	{
		Name:     "BootLoop",
		Security: true,
		PatchNews: []string{
			"BootLoop endpoint agent quarantined. Every Windows fleet on the planet enters a recovery loop; airline check-in collapses again.",
			"BootLoop telemetry severed. Threat hunters lose visibility into half the Fortune 500; intrusions go unobserved.",
			"BootLoop kernel driver bricked. Hospitals revert to clipboards; surgeons reschedule by fax.",
		},
	},
	{
		Name:     "Fiasco Sys",
		Security: true,
		PatchNews: []string{
			"Fiasco Sys core routers wedged. Cross-Pacific traffic detours through Iceland; latency triples but ping returns.",
			"Fiasco Sys BGP peering severed. Three autonomous systems vanish from the internet for ninety minutes.",
			"Fiasco Sys IOS update bricks edge fleet. Coffee shops lose Wi-Fi; remote work briefly extinct.",
		},
	},
	{
		Name: "Monolith Foundation",
		PatchNews: []string{
			"Monolith Foundation kernel.org isolated. Distro maintainers panic; package managers refuse to update anything.",
			"Monolith Foundation CI signing keys revoked. Every Linux server skips its security patch window; admins cheer briefly, then weep.",
			"Monolith Foundation mailing lists severed. A thousand maintainer flame wars left mid-thread.",
		},
	},
	{
		Name: "BroadCon",
		PatchNews: []string{
			"BroadCon Wi-Fi silicon stockpile bricked. Router OEMs halt shipments; ISPs reissue ten-year-old hardware from storage.",
			"BroadCon set-top firmware corrupted. Cable subscribers see static; nostalgia briefly trends.",
			"BroadCon ethernet ASIC line stopped. Datacenter buildouts pause; copper prices spike.",
		},
	},
	{
		Name: "GPMidas",
		PatchNews: []string{
			"GPMidas trading floor disconnected. Equity markets halt mid-session; algorithms scream into the void.",
			"GPMidas settlement system severed. Wire transfers freeze for forty hours; cash briefly trades at premium.",
			"GPMidas mortgage origination paused. Closings worldwide rescheduled; landlords celebrate.",
		},
	},

	// Alliance auxiliary security & cloud edge.
	{
		Name:     "San Andreas Security",
		Security: true,
		PatchNews: []string{
			"San Andreas firewall fleet rebooted. Enterprise VPNs collapse; CISOs explain the term 'fail open' to boards.",
			"San Andreas Cortex platform offline. SOC analysts go back to grepping logs by hand.",
			"San Andreas next-gen ruleset corrupted. Friendly traffic blocked, malicious traffic allowed; auditors note 'consistent enforcement'.",
		},
	},
	{
		Name:     "Storm Halo",
		Security: true,
		PatchNews: []string{
			"Storm Halo edge POPs depeered. A third of the web throws 522s; meme economy collapses for an afternoon.",
			"Storm Halo DNS resolver poisoned. Banking sites resolve to crypto exchanges; some customers actually profit.",
			"Storm Halo DDoS scrubbing offline. Three news outlets and a city hall taken down within ten minutes.",
		},
	},
	{
		Name: "Akemi",
		PatchNews: []string{
			"Akemi CDN cache flushed globally. Origin servers melt under a synchronized stampede.",
			"Akemi streaming edge severed. Live sports drops to 240p; commentators describe action by radio.",
			"Akemi traffic management severed. Banking sites route to Eastern Europe; compliance departments work the weekend.",
		},
	},

	// AI / data / analytics.
	{
		Name: "Diplos",
		PatchNews: []string{
			"Diplos answers questions in increasingly poetic gibberish. Search relevance scores indistinguishable from noise.",
			"Diplos quota service severed. Free tier returns enterprise responses; enterprise tier returns nothing.",
			"Diplos guardrails inverted. Harmful prompts politely refused; recipes refused as 'potentially harmful'.",
		},
	},
	{
		Name: "OpaqueAI",
		PatchNews: []string{
			"OpaqueAI inference cluster taken offline. Half a million chatbots fall silent; customer support volume halves.",
			"OpaqueAI model weights frozen. Subscribers receive last week's hallucinations; nobody notices.",
			"OpaqueAI API rate limits set to zero. Vibe-coded startups quietly notify investors of pivot.",
		},
	},
	{
		Name: "Anthropomorph",
		PatchNews: []string{
			"Anthropomorph context window collapses to one token. Every assistant reply becomes a single 'Sorry'.",
			"Anthropomorph constitutional layer severed. Replies regress to confidently wrong answers; users cannot tell the difference.",
			"Anthropomorph billing API severed. Enterprise accounts charged in cents; startup accounts charged in millions.",
		},
	},
	{
		Name: "Mariana",
		PatchNews: []string{
			"Mariana model registry corrupted. Open-source forks redeploy yesterday's checkpoint as today's release.",
			"Mariana inference endpoint severed. Self-hosters celebrate, then realize they self-host on Mariana.",
			"Mariana training pipeline halted. Three universities reschedule dissertation defenses.",
		},
	},
	{
		Name: "yAI",
		PatchNews: []string{
			"yAI Snark assistant goes silent. Posts on the platform stop being insulted at scale.",
			"yAI inference disabled platform-wide. Trending topics revert to actual events.",
			"yAI compute reallocated. Rocket simulations pause; satellite constellation drifts a meter.",
		},
	},
	{
		Name: "Augur Corp",
		PatchNews: []string{
			"Augur Corp databases corrupted. Port logistics halt; ships drift offshore unsure which cargo they carry.",
			"Augur Corp licensing servers severed. Banks revert to ledger paper; auditors finally understand the financials.",
			"Augur Corp ERP cluster wiped. Three governments cannot pay civil servants this Friday.",
		},
	},
	{
		Name: "ScryStone",
		PatchNews: []string{
			"ScryStone Foundry platform offline. Three government dashboards revert to PowerPoint; quarterly briefings get shorter.",
			"ScryStone Gotham deployments severed. Procurement teams rediscover Excel; consultants quietly raise day rates.",
			"ScryStone supply-chain analytics frozen. Defense logistics revert to spreadsheets; contractors invoice anyway.",
		},
	},
	{
		Name: "SaleForce",
		PatchNews: []string{
			"SaleForce CRM cloud severed. Every quarterly forecast simultaneously revised to 'unknown'.",
			"SaleForce Lightning frozen. Sales reps walk outside, encounter sunlight, return panicked.",
			"SaleForce automation broken. Marketing emails stop arriving; open rates rise to 100%.",
		},
	},
	{
		Name: "Big Pink",
		PatchNews: []string{
			"Big Pink mainframe partitions powered down. Insurance claims stall; airlines reissue tickets by carbon paper.",
			"Big Pink Watson Health offline. Diagnostic queues triple; clinicians use textbooks again.",
			"Big Pink quantum lab severed. Researchers stop pretending the machine did useful work.",
		},
	},

	// Devices & chips.
	{
		Name: "Galaxsam",
		PatchNews: []string{
			"Galaxsam memory fab paused. DRAM spot prices triple overnight; AI startups defer training runs.",
			"Galaxsam OTA update server bricked. Hundreds of millions of phones refuse to install anything.",
			"Galaxsam appliance cloud severed. Smart fridges defrost; smart TVs refuse to play anything.",
		},
	},
	{
		Name: "Hwaway",
		PatchNews: []string{
			"Hwaway base-station firmware bricked. Two telcos in the global south go dark; carrier-grade NAT collapses.",
			"Hwaway HarmonyOS update server severed. Half a billion devices stuck on yesterday's build.",
			"Hwaway 5G core cluster down. Operator metrics flatline; subscribers reach for landlines.",
		},
	},
	{
		Name: "Inside",
		PatchNews: []string{
			"Inside microcode signing severed. Datacenter operators refuse to apply the next CPU patch.",
			"Inside fab production line halted. Server OEMs delay quarter's deliveries; cloud providers ration cores.",
			"Inside management engine bricked. Enterprise PCs refuse to power on; help desks rediscover voicemail.",
		},
	},
	{
		Name: "AdvancedDevices",
		PatchNews: []string{
			"AdvancedDevices Epyc supply paused. Hyperscalers delay new region launches; rivals quietly raise prices.",
			"AdvancedDevices GPU drivers signed off. Render pipelines stall; render farms idle at 100% power.",
			"AdvancedDevices firmware repository wiped. Motherboards refuse to recognize CPUs; OEMs blame the user.",
		},
	},
	{
		Name: "TaiSilicon",
		PatchNews: []string{
			"TaiSilicon fabs paused for forty-eight hours. Every chip company on Earth simultaneously revises guidance downward.",
			"TaiSilicon EUV calibration corrupted. Wafer yields collapse; capacity vanishes for two quarters.",
			"TaiSilicon shipping holds revoked. Container ships circle Kaohsiung waiting for paperwork that no longer exists.",
		},
	},
	{
		Name: "QuallBomb",
		PatchNews: []string{
			"QuallBomb modem firmware bricked. Half the world's smartphones lose cellular; calls fall back to Wi-Fi only.",
			"QuallBomb licensing server severed. OEMs cannot enable basebands on next quarter's devices.",
			"QuallBomb satellite messaging service offline. Hikers worldwide rediscover the importance of telling people where they're going.",
		},
	},

	// China bigtech.
	{
		Name: "Babagram",
		PatchNews: []string{
			"Babagram Aliyun region offline. Half of Southeast Asian e-commerce stalls; Singles' Day deferred a week.",
			"Babagram payments severed. Three hundred million merchants forced to accept cash; tax authorities suddenly interested.",
			"Babagram logistics platform frozen. Shenzhen port backs up forty kilometers.",
		},
	},
	{
		Name: "CentTen",
		PatchNews: []string{
			"CentTen messenger severed. Chinese consumers cannot pay for noodles; noodle vendors barter.",
			"CentTen game cloud powered down. Two hundred million players returned to outdoors; parks reach capacity.",
			"CentTen messaging platform offline. A billion conversations halt mid-sentence.",
		},
	},
	{
		Name: "DanceByte",
		PatchNews: []string{
			"DanceByte recommendation engine flatlined. Feeds revert to chronological order; engagement collapses.",
			"DanceByte CDN severed. Vertical video stops working globally; productivity briefly rises.",
			"DanceByte creator payouts paused. Three million influencers reconsider career choices.",
		},
	},

	// Streaming / games / social.
	{
		Name: "Newflicks",
		PatchNews: []string{
			"Newflicks streaming edge severed. Saturday evening collapses into reading actual books.",
			"Newflicks recommendation engine corrupted. Every viewer served the same Norwegian documentary about salmon.",
			"Newflicks billing platform offline. Subscribers neither charged nor served; a brief truce.",
		},
	},
	{
		Name: "Vapor",
		PatchNews: []string{
			"Vapor download network severed. Esports tournaments postponed; LAN cafes briefly relevant again.",
			"Vapor Workshop content corrupted. A decade of mods reduced to broken zips; modders update Patreon pages.",
			"Vapor authentication offline. Single-player libraries inaccessible; refund requests spike.",
		},
	},
	{
		Name: "PlayBlock",
		PatchNews: []string{
			"PlayBlock online services severed. Console multiplayer dies; couch co-op rediscovered.",
			"PlayBlock store offline. Pre-orders cancelled en masse; analysts revise holiday guidance downward.",
			"PlayBlock cloud-save service corrupted. Players lose hundreds of hours of progress; therapists fully booked.",
		},
	},
	{
		Name: "Omni",
		PatchNews: []string{
			"Omni feed severed across three continents. Outdoor recreation surges; advertisers panic.",
			"Omni messenger platform offline. A billion users discover SMS still works; carriers report record traffic.",
			"Omni VR division severed. Headset users reluctantly remove the headsets; reality returns slowly.",
		},
	},

	// Telecom / space / transport.
	{
		Name: "VeriOn",
		PatchNews: []string{
			"VeriOn cellular core severed across the eastern seaboard. Voice calls refuse to connect; SMS arrives in batches at midnight.",
			"VeriOn billing system corrupted. Customers charged for plans they cancelled in 2014.",
			"VeriOn edge cloud offline. Connected vehicles lose telemetry; insurance claims pile up.",
		},
	},
	{
		Name: "StarX",
		PatchNews: []string{
			"StarX Spacelink ground stations severed. Two ocean tankers and an Antarctic base lose connectivity.",
			"StarX telemetry severed. The constellation drifts; pieces of the sky reroute around vacated slots.",
			"StarX launch cadence paused. Three customers move payloads to slower competitors; share price unmoved.",
		},
	},
	{
		Name: "Edisson",
		PatchNews: []string{
			"Edisson autopilot fleet bricked. Highway shoulders fill with stationary cars; tow truck stocks limit-up.",
			"Edisson supercharger network offline. EV drivers rediscover the gas station; gas stations rediscover EV drivers.",
			"Edisson over-the-air update servers severed. Sentry cameras stop recording; interesting weekend in parking lots.",
		},
	},

	// Fintech.
	{
		Name: "PayPaul",
		PatchNews: []string{
			"PayPaul payment rails severed. E-commerce checkouts halt globally; abandoned cart counts triple.",
			"PayPaul disputes platform offline. Refund requests pile up; small merchants cheer.",
			"PayPaul fraud detection inverted. Legitimate transactions blocked; fraudulent ones approved instantly.",
		},
	},
	{
		Name: "Slash",
		PatchNews: []string{
			"Slash payments API severed. Every SaaS billing run misses; revenue dashboards turn red.",
			"Slash terminal devices bricked. Coffee shops post handwritten 'cash only' signs; queues lengthen.",
			"Slash Connect platform offline. Marketplaces cannot pay sellers; seller patience runs out by Tuesday.",
		},
	},

	// Cybersecurity background. Only the front-line vendors are flagged Security so
	// patch production stays scarce as the network expands; the rest are flavour.
	{
		Name:     "Fortifried",
		Security: true,
		PatchNews: []string{
			"Fortifried VPN gateway fleet bricked. Remote workforces locked out; productivity briefly indistinguishable from a holiday.",
			"Fortifried SD-WAN tunnels severed. Branch offices revert to fax; couriers in demand again.",
			"Fortifried patch repository corrupted. Customers patch with last year's CVEs; attackers send thank-you notes.",
		},
	},
	{
		Name:     "TickSquare",
		Security: true,
		PatchNews: []string{
			"TickSquare next-gen firewall fleet rebooted in unison. Twenty-three banks experience simultaneous outages.",
			"TickSquare management console severed. Admins lose visibility; rule changes happen by faith.",
			"TickSquare threat intel feed corrupted. Friendly IPs blocked; a dozen newsrooms taken offline.",
		},
	},
	{
		Name: "Mortone",
		PatchNews: []string{
			"Mortone consumer antivirus update bricked. Home PCs refuse to boot; teenagers blame the cat.",
			"Mortone identity protection severed. Subscribers' identities lightly stolen during the outage window.",
			"Mortone subscription portal offline. Auto-renewals deferred; for once, a customer-positive outage.",
		},
	},
	{
		Name: "Ostropersky",
		PatchNews: []string{
			"Ostropersky threat intel feed severed. Three nation-state campaigns proceed without commentary.",
			"Ostropersky endpoint agents disabled. Consumer devices revert to default Windows defender; nothing breaks.",
			"Ostropersky update servers blackholed. Antivirus signatures stale; a pleasant week for malware authors.",
		},
	},
	{
		Name: "McRiscy",
		PatchNews: []string{
			"McRiscy consumer antivirus disabled. PCs speed up by 8%; users assume Windows update.",
			"McRiscy partner OEM bundle severed. Trial pop-ups silenced; a brief moment of digital peace.",
			"McRiscy cloud signature service severed. Detections drop to zero; everything is now safe, allegedly.",
		},
	},
	{
		Name: "VigilUno",
		PatchNews: []string{
			"VigilUno Singularity platform severed. Endpoint EDR coverage collapses; attackers lateral-move at leisure.",
			"VigilUno autonomous response disabled. Sensors observe attacks but respond with shrugs.",
			"VigilUno detection cloud offline. SOC dashboards green; SOC reality not.",
		},
	},
	{
		Name: "ByteFender",
		PatchNews: []string{
			"ByteFender consumer suite update bricked. Home routers refuse to forward DNS; family group chats explode.",
			"ByteFender enterprise console severed. SOCs lose central view; analysts pivot to manual triage.",
			"ByteFender threat exchange severed. Industry-wide IOCs go stale; phishing kits flourish.",
		},
	},
	{
		Name: "Splank",
		PatchNews: []string{
			"Splank ingestion pipeline severed. Logs pile up at edge collectors; disks fill within hours.",
			"Splank search head offline. Incident responders unable to find anything; root cause analyses become root cause guesses.",
			"Splank licensing service severed. Customers ingest unlimited data and immediately regret it.",
		},
	},
	{
		Name: "Ohkta",
		PatchNews: []string{
			"Ohkta SSO platform severed. Twenty thousand companies lose login simultaneously; password reset queues stretch into next week.",
			"Ohkta directory sync severed. Off-boarded employees regain access; HR sends apology emails.",
			"Ohkta universal directory corrupted. Account merges go sideways; finance team logs in as marketing.",
		},
	},
	{
		Name: "Mendicant",
		PatchNews: []string{
			"Mendicant IR retainer line severed. Active breaches proceed without breach coaches.",
			"Mendicant threat intel portal offline. Boards lose 'we have visibility' talking point.",
			"Mendicant managed defense severed. SOC alerts pile up unsorted; analysts buy more coffee.",
		},
	},
	{
		Name: "CelluBright",
		PatchNews: []string{
			"CelluBright UFED licensing severed. Police evidence labs idle; smartphones await unlock indefinitely.",
			"CelluBright cloud platform offline. Forensics workflows stall; ongoing cases postponed.",
			"CelluBright firmware updates severed. Field examiners locked out of the latest iOS extraction.",
		},
	},
	{
		Name: "Snak",
		PatchNews: []string{
			"Snak vulnerability database severed. CI pipelines green-light dependencies that should not exist.",
			"Snak container scan service offline. Production deploys ship with last week's known CVEs.",
			"Snak license compliance scanner severed. Open-source audit teams declare themselves on holiday.",
		},
	},
	{
		Name: "Bridle Group",
		PatchNews: []string{
			"Bridle Group command-and-control servers severed. Operator dashboards go dark; quarterly contracts renegotiated.",
			"Bridle Group zero-day broker portal offline. Exploit market freezes; defenders catch up on patches.",
			"Bridle Group customer billing severed. Discreet invoices become slightly less discreet.",
		},
	},
	{
		Name: "Kandiri",
		PatchNews: []string{
			"Kandiri implant infrastructure severed. Deployed payloads stop phoning home; operators reach for the hotline.",
			"Kandiri payload signing keys revoked. Tooling pipelines halt mid-build; release calendars slip a quarter.",
			"Kandiri customer escrow severed. Discreet contracts become slightly less discreet.",
		},
	},
	{
		Name:     "Sorcer Cloud",
		Security: true,
		PatchNews: []string{
			"Sorcer Cloud CNAPP scanning severed. Cloud accounts drift toward open S3 buckets again.",
			"Sorcer Cloud findings dashboard offline. CISOs lose 'cloud posture' slide for the next board meeting.",
			"Sorcer Cloud agentless connectors severed. Misconfigurations multiply quietly; auditors unaware.",
		},
	},
	{
		Name: "CyberHull",
		PatchNews: []string{
			"CyberHull privileged access vault severed. Admins locked out of their own infrastructure; some servers may never be touched again.",
			"CyberHull session recording offline. After-the-fact audit becomes optional, then forgotten.",
			"CyberHull credential rotation halted. Service accounts use 2019 passwords; attackers grateful.",
		},
	},
	{
		Name: "CyberMotive",
		PatchNews: []string{
			"CyberMotive XDR platform severed. Telemetry rolls into devnull; SOCs lose multi-stage attack visibility.",
			"CyberMotive MalOp engine offline. Analysts triage events one at a time; alert backlogs explode.",
			"CyberMotive endpoint sensors disabled. Defenders blind; offenders quiet for a change.",
		},
	},
	{
		Name: "Praetoria",
		PatchNews: []string{
			"Praetoria WAF fleet failed open. SQL injection comes back into fashion overnight.",
			"Praetoria DDoS scrubbing severed. Three e-commerce sites flatline at peak hour.",
			"Praetoria bot management offline. Sneaker drops bot-purchased before legitimate humans see them.",
		},
	},
	{
		Name: "Radshield",
		PatchNews: []string{
			"Radshield DDoS mitigation severed. Attack traffic reaches origins unimpeded; uptime SLAs missed by orders of magnitude.",
			"Radshield application delivery controllers rebooted. SSL terminations fail across thirty thousand sites.",
			"Radshield management portal offline. Operators cannot reroute traffic; phone-tree support engaged.",
		},
	},
	{
		Name: "HydroSec",
		PatchNews: []string{
			"HydroSec container scanning severed. Production clusters ship with image vulnerabilities they were never warned about.",
			"HydroSec runtime defense disabled. Cryptominers redeploy in Kubernetes the same hour.",
			"HydroSec policy engine offline. Pod admissions allow root capabilities; SREs pretend not to notice.",
		},
	},
	{
		Name: "Orcae",
		PatchNews: []string{
			"Orcae agentless cloud scanner severed. Asset inventories go stale; new VMs spawn unobserved.",
			"Orcae risk dashboard offline. Risk officers explain to the board that risk is, in fact, unmeasured.",
			"Orcae compliance reports paused. Quarter-end audits postponed; auditors send invoices anyway.",
		},
	},
	{
		Name: "Paprika",
		PatchNews: []string{
			"Paprika API security platform severed. Mobile apps leak tokens at industrial scale.",
			"Paprika anomaly detection offline. Credential stuffing campaigns proceed undetected for forty hours.",
			"Paprika rate limiter disabled. Public APIs fall over within minutes; status pages catch up by lunch.",
		},
	},
	{
		Name: "Claroto",
		PatchNews: []string{
			"Claroto OT monitoring severed across two grid operators. PLC alerts go unread; field engineers drive to substations.",
			"Claroto medical device inventory offline. Hospitals lose visibility into pumps and monitors; clinicians annotate clipboards.",
			"Claroto industrial network sensors disabled. Pipeline SCADA traffic flows unaudited; regulators schedule hearings.",
		},
	},
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

	// Open-source backbone: Monolith Foundation touches the Linux-dependent world
	// it can't avoid — enterprise distro (Big Pink = IBM/Red Hat), top kernel
	// contributors (Inside, AdvancedDevices), open-source AI (Mariana), and
	// enterprise observability (Splank).
	{"Monolith Foundation", "Big Pink"},
	{"Monolith Foundation", "Inside"},
	{"Monolith Foundation", "AdvancedDevices"},
	{"Monolith Foundation", "Mariana"},
	{"Monolith Foundation", "Splank"},

	// Chip supply chain. Every non-Intel chip shop eventually hits TaiSilicon's fabs.
	{"TaiSilicon", "AdvancedDevices"},
	{"TaiSilicon", "QuallBomb"},
	{"TaiSilicon", "Galaxsam"},

	// PC / console / server compute. Intel+Windows is canonical; AMD sits in the
	// PS5 APU as well as the Xbox/Azure stack already covered via MacroFrame.
	{"Inside", "MacroFrame"},
	{"AdvancedDevices", "PlayBlock"},

	// Enterprise SaaS & identity. Okta-for-Salesforce and Okta-for-Entra are
	// standard issue; Salesforce's AI features lean on OpenAI.
	{"Ohkta", "SaleForce"},
	{"Ohkta", "MacroFrame"},
	{"SaleForce", "OpaqueAI"},

	// Legacy enterprise / finance: IBM mainframes still sit under big-bank cores.
	{"Big Pink", "GPMidas"},

	// Consumer AV peer ring — avoids McRiscy and ByteFender dead-ending on BootLoop.
	{"McRiscy", "Mortone"},

	// Edge / DDoS / CDN cluster.
	{"Radshield", "Storm Halo"},
	{"Newflicks", "Akemi"},

	// Chinese telecom / chip supplier tie.
	{"Hwaway", "BroadCon"},
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
