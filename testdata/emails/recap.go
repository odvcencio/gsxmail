package emails

// RecapProps fills DraftRecap, the design spec's section 6.5 worked
// example ("the N7 recap 'YOUR HAUL'"): a recipient-personalized stat
// table built from an <Each> over a slice of struct rows.
type RecapProps struct {
	League         string
	Code           string
	Tagline        string
	PickCountLabel string
	Lede           string
	HaulHeader     []string
	Haul           []HaulRow
	HasAutoPicks   bool
	AutoPickNote   string
	BoardURL       string
	FooterNote     string
}

// HaulRow is one Haul entry: a StatRow's cells, and whether it is the
// recipient's marked ("keystone") pick.
type HaulRow struct {
	Cells      []string
	IsKeystone bool
}
