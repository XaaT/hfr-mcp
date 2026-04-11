package hfr

import "html"

// Category represents an HFR forum category
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// SubCategory represents an HFR forum subcategory
type SubCategory struct {
	ID       int    `json:"id"`
	CatID    int    `json:"cat_id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	CatName  string `json:"cat_name"`
}

// Categories returns the hardcoded list of HFR categories.
// HFR categories rarely change — this is the reliable source.
func Categories() []Category {
	return []Category{
		{1, "Hardware", "Hardware"},
		{16, "Hardware - Périphériques", "HardwarePeripheriques"},
		{15, "Ordinateurs portables", "OrdinateursPortables"},
		{2, "Overclocking, Cooling & Modding", "OverclockingCoolingModding"},
		{30, "Electronique, domotique, DIY", "electroniquedomotiquediy"},
		{23, "Technologies Mobiles", "gsmgpspda"},
		{25, "Apple", "apple"},
		{3, "Video & Son", "VideoSon"},
		{14, "Photo numérique", "Photonumerique"},
		{5, "Jeux Video", "JeuxVideo"},
		{4, "Windows & Software", "WindowsSoftware"},
		{22, "Réseaux grand public / SoHo", "reseauxpersosoho"},
		{21, "Systèmes & Réseaux Pro", "systemereseauxpro"},
		{11, "Linux et OS Alternatifs", "OSAlternatifs"},
		{10, "Programmation", "Programmation"},
		{12, "Graphisme", "Graphisme"},
		{6, "Achats & Ventes", "AchatsVentes"},
		{8, "Emploi & Etudes", "EmploiEtudes"},
		{13, "Discussions", "Discussions"},
		{24, "Blabla - Divers", "Blabla-Divers-back-to-life"},
	}
}

// SubCategories returns the hardcoded list of HFR subcategories.
func SubCategories() []SubCategory {
	h := html.UnescapeString
	_ = h
	subs := []SubCategory{
		// Hardware (1)
		{108, 1, "Carte mère", "carte-mere", "Hardware"},
		{534, 1, "Mémoire", "Memoire", "Hardware"},
		{533, 1, "Processeur", "Processeur", "Hardware"},
		{109, 1, "Carte graphique", "2D-3D", "Hardware"},
		{466, 1, "Boitier", "Boitier", "Hardware"},
		{532, 1, "Alimentation", "Alimentation", "Hardware"},
		{110, 1, "Disque dur", "HDD", "Hardware"},
		{531, 1, "Disque SSD", "SSD", "Hardware"},
		{467, 1, "CD/DVD/BD", "lecteur-graveur", "Hardware"},
		{507, 1, "Mini PC", "minipc", "Hardware"},
		{252, 1, "Bench", "Bench", "Hardware"},
		{253, 1, "Matériels & problèmes divers", "Materiels-problemes-divers", "Hardware"},
		{481, 1, "Conseil d'achat", "conseilsachats", "Hardware"},
		{546, 1, "HFR", "hfr", "Hardware"},
		{578, 1, "Actus", "actualites", "Hardware"},
		// Hardware - Périphériques (16)
		{451, 16, "Ecran", "Ecran", "Hardware - Périphériques"},
		{452, 16, "Imprimante", "Imprimante", "Hardware - Périphériques"},
		{453, 16, "Scanner", "Scanner", "Hardware - Périphériques"},
		{462, 16, "Webcam / Caméra IP", "webcam-camera-ip", "Hardware - Périphériques"},
		{454, 16, "Clavier / Souris", "Clavier-Souris", "Hardware - Périphériques"},
		{455, 16, "Joys", "Joys", "Hardware - Périphériques"},
		{530, 16, "Onduleur", "Onduleur", "Hardware - Périphériques"},
		{456, 16, "Divers", "Divers", "Hardware - Périphériques"},
		// Ordinateurs portables (15)
		{448, 15, "Portable", "portable", "Ordinateurs portables"},
		{512, 15, "Ultraportable", "Ultraportable", "Ordinateurs portables"},
		{516, 15, "Transportable", "Transportable", "Ordinateurs portables"},
		{520, 15, "Netbook", "Netbook", "Ordinateurs portables"},
		{515, 15, "Composant", "Composant", "Ordinateurs portables"},
		{517, 15, "Accessoire", "Accessoire", "Ordinateurs portables"},
		{513, 15, "Conseils d'achat", "Conseils-d-achat", "Ordinateurs portables"},
		{479, 15, "SAV", "SAV", "Ordinateurs portables"},
		// Overclocking, Cooling & Modding (2)
		{458, 2, "CPU", "CPU", "Overclocking, Cooling & Modding"},
		{119, 2, "GPU", "GPU", "Overclocking, Cooling & Modding"},
		{117, 2, "Air Cooling", "Air-Cooling", "Overclocking, Cooling & Modding"},
		{118, 2, "Water & Xtreme Cooling", "Water-Xtreme-Cooling", "Overclocking, Cooling & Modding"},
		{400, 2, "Silence", "Silence", "Overclocking, Cooling & Modding"},
		{461, 2, "Modding", "Modding", "Overclocking, Cooling & Modding"},
		{121, 2, "Divers", "Divers", "Overclocking, Cooling & Modding"},
		// Electronique, domotique, DIY (30)
		{571, 30, "Conception, dépannage, mods", "conception_depannage_mods", "Electronique, domotique, DIY"},
		{572, 30, "Nano-ordinateur, microcontrôleurs, FPGA", "nano-ordinateur_microcontroleurs_fpga", "Electronique, domotique, DIY"},
		{573, 30, "Domotique et maison connectée", "domotique_maisonconnectee", "Electronique, domotique, DIY"},
		{574, 30, "Mécanique, prototypage", "mecanique_prototypage", "Electronique, domotique, DIY"},
		{575, 30, "Imprimantes 3D", "imprimantes3D", "Electronique, domotique, DIY"},
		{576, 30, "Robotique et modélisme", "robotique_modelisme", "Electronique, domotique, DIY"},
		{577, 30, "Divers", "divers", "Electronique, domotique, DIY"},
		// Technologies Mobiles (23)
		{567, 23, "Autres OS Mobiles", "autres-os-mobiles", "Technologies Mobiles"},
		{510, 23, "Opérateur", "operateur", "Technologies Mobiles"},
		{553, 23, "Téléphone Android", "telephone-android", "Technologies Mobiles"},
		{554, 23, "Téléphone Windows Phone", "telephone-windows-phone", "Technologies Mobiles"},
		{529, 23, "Téléphone", "telephone", "Technologies Mobiles"},
		{540, 23, "Tablette", "tablette", "Technologies Mobiles"},
		{550, 23, "Android", "android", "Technologies Mobiles"},
		{551, 23, "Windows Phone", "windows-phone", "Technologies Mobiles"},
		{509, 23, "GPS / PDA", "GPS-PDA", "Technologies Mobiles"},
		{561, 23, "Accessoires", "accessoires", "Technologies Mobiles"},
		// Apple (25)
		{522, 25, "Mac OS X", "Mac-OS-X", "Apple"},
		{528, 25, "Applications", "Applications", "Apple"},
		{523, 25, "Mac", "Mac", "Apple"},
		{524, 25, "Macbook", "Macbook", "Apple"},
		{525, 25, "Iphone & Ipod", "Iphone-amp-Ipod", "Apple"},
		{535, 25, "Ipad", "Ipad", "Apple"},
		{526, 25, "Périphériques", "Peripheriques", "Apple"},
		// Video & Son (3)
		{130, 3, "HiFi & Home Cinema", "HiFi-HomeCinema", "Video & Son"},
		{129, 3, "Matériel", "Materiel", "Video & Son"},
		{131, 3, "Traitement Audio", "Traitement-Audio", "Video & Son"},
		{134, 3, "Traitement Vidéo", "Traitement-Video", "Video & Son"},
		// Photo numérique (14)
		{442, 14, "Appareil", "Appareil", "Photo numérique"},
		{519, 14, "Objectif", "Objectif", "Photo numérique"},
		{443, 14, "Accessoire", "Accessoire", "Photo numérique"},
		{444, 14, "Photos", "Photos", "Photo numérique"},
		{445, 14, "Technique", "Technique", "Photo numérique"},
		{446, 14, "Logiciels & Retouche", "Logiciels-Retouche", "Photo numérique"},
		{447, 14, "Argentique", "Argentique", "Photo numérique"},
		{476, 14, "Concours", "Concours", "Photo numérique"},
		{478, 14, "Galerie Perso", "Galerie-Perso", "Photo numérique"},
		{457, 14, "Divers", "Divers", "Photo numérique"},
		// Jeux Video (5)
		{249, 5, "PC", "PC", "Jeux Video"},
		{250, 5, "Consoles", "Consoles", "Jeux Video"},
		{251, 5, "Achat & Ventes", "Achat-Ventes", "Jeux Video"},
		{412, 5, "Teams & LAN", "Teams-LAN", "Jeux Video"},
		{413, 5, "Tips & Dépannage", "Tips-Depannage", "Jeux Video"},
		{579, 5, "Réalité virtuelle", "VR-Realite-Virtuelle", "Jeux Video"},
		{569, 5, "Mobiles", "mobiles", "Jeux Video"},
		// Windows & Software (4)
		{580, 4, "Win 11", "windows-11", "Windows & Software"},
		{570, 4, "Win 10", "windows-10", "Windows & Software"},
		{555, 4, "Win 8", "windows-8", "Windows & Software"},
		{521, 4, "Win 7", "windows-7-seven", "Windows & Software"},
		{505, 4, "Win Vista", "windows-vista", "Windows & Software"},
		{406, 4, "Win NT/2K/XP", "windows-nt-2k-xp", "Windows & Software"},
		{504, 4, "Win 9x/Me", "windows-9x-me", "Windows & Software"},
		{437, 4, "Sécurité", "Securite", "Windows & Software"},
		{506, 4, "Virus/Spywares", "Virus-Spywares", "Windows & Software"},
		{435, 4, "Stockage/Sauvegarde", "Stockage-Sauvegarde", "Windows & Software"},
		{407, 4, "Logiciels", "Logiciels", "Windows & Software"},
		{438, 4, "Tutoriels", "Tutoriels", "Windows & Software"},
		// Réseaux grand public / SoHo (22)
		{496, 22, "FAI", "FAI", "Réseaux grand public / SoHo"},
		{503, 22, "Réseaux", "Reseaux", "Réseaux grand public / SoHo"},
		{497, 22, "Sécurité", "Routage-et-securite", "Réseaux grand public / SoHo"},
		{498, 22, "WiFi et CPL", "WiFi-et-CPL", "Réseaux grand public / SoHo"},
		{499, 22, "Hébergement", "Hebergement", "Réseaux grand public / SoHo"},
		{500, 22, "Tel / TV sur IP", "Tel-TV-sur-IP", "Réseaux grand public / SoHo"},
		{501, 22, "Chat, visio et voix", "Chat-visio-et-voix", "Réseaux grand public / SoHo"},
		{502, 22, "Tutoriels", "Tutoriels", "Réseaux grand public / SoHo"},
		// Systèmes & Réseaux Pro (21)
		{487, 21, "Réseaux", "Reseaux", "Systèmes & Réseaux Pro"},
		{488, 21, "Sécurité", "Securite", "Systèmes & Réseaux Pro"},
		{489, 21, "Télécom", "Telecom", "Systèmes & Réseaux Pro"},
		{491, 21, "Infrastructures serveurs", "Infrastructures-serveurs", "Systèmes & Réseaux Pro"},
		{492, 21, "Stockage", "Stockage", "Systèmes & Réseaux Pro"},
		{493, 21, "Logiciels d'entreprise", "Logiciels-entreprise", "Systèmes & Réseaux Pro"},
		{494, 21, "Management du SI", "Management-SI", "Systèmes & Réseaux Pro"},
		{544, 21, "Poste de travail", "poste-de-travail", "Systèmes & Réseaux Pro"},
		// Linux et OS Alternatifs (11)
		{209, 11, "Codes et scripts", "Codes-scripts", "Linux et OS Alternatifs"},
		{205, 11, "Débats", "Debats", "Linux et OS Alternatifs"},
		{420, 11, "Divers", "Divers-2", "Linux et OS Alternatifs"},
		{472, 11, "Hardware", "Hardware-2", "Linux et OS Alternatifs"},
		{204, 11, "Installation", "Installation", "Linux et OS Alternatifs"},
		{208, 11, "Logiciels", "Logiciels-2", "Linux et OS Alternatifs"},
		{207, 11, "Multimédia", "Multimedia", "Linux et OS Alternatifs"},
		{206, 11, "Réseaux et sécurité", "reseaux-securite", "Linux et OS Alternatifs"},
		// Programmation (10)
		{381, 10, "Ada", "Ada", "Programmation"},
		{382, 10, "Algo", "Algo", "Programmation"},
		{562, 10, "Android", "Android", "Programmation"},
		{518, 10, "API Win32", "API-Win32", "Programmation"},
		{384, 10, "ASM", "ASM", "Programmation"},
		{383, 10, "ASP", "ASP", "Programmation"},
		{565, 10, "BI/Big Data", "BI-Big-Data", "Programmation"},
		{440, 10, "C", "C", "Programmation"},
		{405, 10, "C#/.NET managed", "C-NET-managed", "Programmation"},
		{386, 10, "C++", "C-2", "Programmation"},
		{391, 10, "Delphi/Pascal", "Delphi-Pascal", "Programmation"},
		{473, 10, "Flash/ActionScript", "Flash-ActionScript", "Programmation"},
		{389, 10, "HTML/CSS", "HTML-CSS-Javascript", "Programmation"},
		{563, 10, "iOS", "iOS", "Programmation"},
		{390, 10, "Java", "Java", "Programmation"},
		{566, 10, "Javascript/Node.js", "Javascript-Node-js", "Programmation"},
		{484, 10, "Langages fonctionnels", "Langages-fonctionnels", "Programmation"},
		{392, 10, "Perl", "Perl", "Programmation"},
		{393, 10, "PHP", "PHP", "Programmation"},
		{394, 10, "Python", "Python", "Programmation"},
		{483, 10, "Ruby/Rails", "Ruby-Rails", "Programmation"},
		{404, 10, "Shell/Batch", "Shell-Batch", "Programmation"},
		{395, 10, "SQL/NoSQL", "SGBD-SQL", "Programmation"},
		{396, 10, "VB/VBA/VBS", "VB-VBA-VBS", "Programmation"},
		{564, 10, "Windows Phone", "Windows-Phone", "Programmation"},
		{439, 10, "XML/XSL", "XML-XSL", "Programmation"},
		{388, 10, "Divers", "Divers", "Programmation"},
		// Graphisme (12)
		{475, 12, "Cours", "Cours", "Graphisme"},
		{469, 12, "Galerie", "Galerie", "Graphisme"},
		{227, 12, "Infographie 2D", "Infographie-2D", "Graphisme"},
		{470, 12, "PAO / Desktop Publishing", "PAO-Desktop-Publishing", "Graphisme"},
		{228, 12, "Infographie 3D", "Infographie-3D", "Graphisme"},
		{402, 12, "Web design", "Webdesign", "Graphisme"},
		{441, 12, "Arts traditionnels", "Arts-traditionnels", "Graphisme"},
		{229, 12, "Concours", "Concours", "Graphisme"},
		{230, 12, "Ressources", "Ressources", "Graphisme"},
		{231, 12, "Divers", "Divers", "Graphisme"},
		// Achats & Ventes (6)
		{169, 6, "Hardware", "Hardware", "Achats & Ventes"},
		{536, 6, "PC Portables", "pc-portables", "Achats & Ventes"},
		{560, 6, "Tablettes", "tablettes", "Achats & Ventes"},
		{171, 6, "Photo", "Photo-Audio-Video", "Achats & Ventes"},
		{537, 6, "Audio, Vidéo", "audio-video", "Achats & Ventes"},
		{173, 6, "Téléphonie", "Telephonie", "Achats & Ventes"},
		{170, 6, "Softs, livres", "Softs-livres", "Achats & Ventes"},
		{174, 6, "Divers", "Divers", "Achats & Ventes"},
		{398, 6, "Avis, estimations", "Avis-estimations", "Achats & Ventes"},
		{416, 6, "Feed-back", "Feedback", "Achats & Ventes"},
		{399, 6, "Règles et coutumes", "Regles-coutumes", "Achats & Ventes"},
		// Emploi & Etudes (8)
		{233, 8, "Marché de l'emploi", "Marche-emploi", "Emploi & Etudes"},
		{235, 8, "Etudes / Orientation", "Etudes-Orientation", "Emploi & Etudes"},
		{234, 8, "Annonces d'emplois", "Annonces-emplois", "Emploi & Etudes"},
		{464, 8, "Feedback sur les entreprises", "Feedback-entreprises", "Emploi & Etudes"},
		{465, 8, "Aide aux devoirs", "Aide-devoirs", "Emploi & Etudes"},
		// Discussions (13)
		{422, 13, "Actualité", "Actualite", "Discussions"},
		{482, 13, "Politique", "politique", "Discussions"},
		{423, 13, "Société", "Societe", "Discussions"},
		{424, 13, "Cinéma", "Cinema", "Discussions"},
		{425, 13, "Musique", "Musique", "Discussions"},
		{426, 13, "Arts & Lecture", "Arts-Lecture", "Discussions"},
		{427, 13, "TV, Radio", "TV-Radio", "Discussions"},
		{428, 13, "Sciences", "Sciences", "Discussions"},
		{429, 13, "Santé", "Sante", "Discussions"},
		{430, 13, "Sports", "Sports", "Discussions"},
		{431, 13, "Auto / Moto", "Auto-Moto", "Discussions"},
		{433, 13, "Cuisine", "Cuisine", "Discussions"},
		{434, 13, "Loisirs", "Loisirs", "Discussions"},
		{557, 13, "Voyages", "voyages", "Discussions"},
		{432, 13, "Vie pratique", "Viepratique", "Discussions"},
	}
	return subs
}

// SubCategoriesForCat returns subcategories for a given category ID.
func SubCategoriesForCat(catID int) []SubCategory {
	var result []SubCategory
	for _, sc := range SubCategories() {
		if sc.CatID == catID {
			result = append(result, sc)
		}
	}
	return result
}
