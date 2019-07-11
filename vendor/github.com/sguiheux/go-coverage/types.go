package coverage

// Parser represents the lcov parser
type Parser struct {
	path string
	mode CoverageMode
}

//Report represents the result of  LcovParser.parse
type Report struct {
	Files            []FileReport `json:"files"`
	TotalLines       int          `json:"total_lines"`
	CoveredLines     int          `json:"covered_lines"`
	TotalFunctions   int          `json:"total_functions"`
	CoveredFunctions int          `json:"covered_functions"`
	TotalBranches    int          `json:"total_branches"`
	CoveredBranches  int          `json:"covered_branches"`
}

// FileReport contains all informations about a file
type FileReport struct {
	Path             string `json:"path"`
	TotalLines       int    `json:"total_lines"`
	CoveredLines     int    `json:"covered_lines"`
	TotalFunctions   int    `json:"total_functions"`
	CoveredFunctions int    `json:"covered_functions"`
	TotalBranches    int    `json:"total_branches"`
	CoveredBranches  int    `json:"covered_branches"`
}

// CoverageMode represents the format of the coverage reprt
type CoverageMode string

const (
	LCOV      CoverageMode = "lcov"
	COBERTURA CoverageMode = "cobertura"
	CLOVER    CoverageMode = "clover"
)
