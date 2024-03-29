package cells

import "github.com/ddsnellings/weaver/variants"

var DefaultVcfQual float64 = 100
var DefaultCellFilter = CellFilterParam{MinGenotypeQuality: 30, MinGenotypeDepth: 10, MinReadAf: 0.2}
var DefaultGlobalFilter = GlobalFilterParam{MinGenotypedFrac: 0.5, MinGenotypesPresent: 0.5, MinCellAf: 0.01}

// CellFilterParam defines the minimum values used for filtering cell.
// These filters are applied on a per-cell basis
type CellFilterParam struct {
	MinGenotypeQuality int     // remove genotype in cell with quality < MinGenotypeQuality // Default 30
	MinGenotypeDepth   int     // remove genotype in cell with read depth < MinGenotypeDepth // Default 10
	MinReadAf          float64 // remove genotype in cell with alternate allele frequency < MinCellAf // Default 0.2
}

// GlobalFilterParam defines minimum values used for filtering cells and variants.
// These filters are applied on a global basis in the order shown below.
type GlobalFilterParam struct {
	MinGenotypedFrac    float64 // remove variants genotyped in < MinGenotypedFrac of cells // Default 0.5
	MinGenotypesPresent float64 // remove cells with < MinGenotypesPresent of valid genotypes present // Default 0.5
	MinCellAf           float64 // remove variants mutated in < MinCellAf of cells // Default 0.01
}

// Apply the global filter to the input data.
// Removes filtered cells and variants and updates the respective Ids.
func (f GlobalFilterParam) Apply(d *Data) {
	ignoreCells := make([]bool, len(d.Cells))
	ignoreVariants := make([]bool, len(d.Variants))
	applyMinGenotypedFrac(d, f, ignoreCells, ignoreVariants)
	applyMinGenotypesPresent(d, f, ignoreCells, ignoreVariants)
	applyMinCellAf(d, f, ignoreCells, ignoreVariants)
	removeFailing(d, ignoreCells, ignoreVariants)
}

// applyMinGenotypedFrac sets ignoredVariants[i] to true if Variants[i] has less than the minimum
// fraction of cells genotyped.
func applyMinGenotypedFrac(d *Data, f GlobalFilterParam, ignoreCells []bool, ignoreVariants []bool) {
	totalCells := float64(len(d.Cells))
	for i := range d.Variants {
		if ignoreVariants[i] {
			continue
		}

		if float64(len(d.Variants[i].CellsGenotyped))/totalCells < f.MinGenotypedFrac {
			ignoreVariants[i] = true
		}
	}
}

// applyMinGenotypesPresent sets ignoredCells[i] to true if Cells[i] has less than the minimum
// fraction of variants present
func applyMinGenotypesPresent(d *Data, f GlobalFilterParam, ignoreCells []bool, ignoreVariants []bool) {
	passingVariants := countFalse(ignoreVariants)
	for i := range d.Cells {
		if ignoreCells[i] {
			continue
		}

		if countPassingCellVar(d.Cells[i].Genotypes, ignoreVariants)/passingVariants < f.MinGenotypesPresent {
			ignoreCells[i] = true
		}
	}
}

// applyMinCellAf sets ignoredVariants[i] to true if Variants[i] has less than the minimum
// fraction of mutant cells present
func applyMinCellAf(d *Data, f GlobalFilterParam, ignoreCells []bool, ignoreVariants []bool) {
	for i := range d.Variants {
		if ignoreVariants[i] {
			continue
		}

		if float64(len(d.Variants[i].CellsMutated))/float64(len(d.Variants[i].CellsGenotyped)) < f.MinCellAf {
			ignoreVariants[i] = true
		}
	}
}

// removeFailing removes all cells and variants which were determined should be ignored
func removeFailing(d *Data, ignoreCells []bool, ignoreVariants []bool) {
	passingCells, passingVariants, newCellIds, newVariantIds := fetchPassingCellsAndVariants(d, ignoreCells, ignoreVariants)
	d.Cells = passingCells
	d.Variants = passingVariants

	totalCells := float64(len(d.Cells))
	totalVariants := float64(len(d.Variants))

	// update fields inside cells
	for i := range d.Cells {
		d.Cells[i].Genotypes = updateCellVar(d.Cells[i].Genotypes, ignoreVariants, newVariantIds)
		d.Cells[i].GenotypesPresent = float64(len(d.Cells[i].Genotypes)) / totalVariants
	}

	// update fields inside variants
	for i := range d.Variants {
		d.Variants[i].CellsGenotyped = updateCellIds(d.Variants[i].CellsGenotyped, ignoreCells, newCellIds)
		d.Variants[i].CellsMutated = updateCellIds(d.Variants[i].CellsMutated, ignoreCells, newCellIds)
		d.Variants[i].GenotypedFrac = float64(len(d.Variants[i].CellsGenotyped)) / totalCells
		d.Variants[i].CellsMutatedFrac = float64(len(d.Variants[i].CellsMutated)) / float64(len(d.Variants[i].CellsGenotyped))
		d.Variants[i].CellAf = getCellAf(d, i)
	}
}

// getCellAf determines the mutant alleles/WT alleles for a single variant
func getCellAf(d *Data, Vid int) float64 {
	var total, mutant int
	for _, cellId := range d.Variants[Vid].CellsGenotyped {
		total += 2 // TODO HEMIZYGOSITY
		mutant += zygosityToInt(d.Cells[cellId].Genotypes[Vid].Genotype)
	}
	return float64(mutant) / float64(total)
}

// zygosityToInt returns the number of alleles mutated for zygosity z
func zygosityToInt(z variants.Zygosity) int {
	switch z {
	case variants.WildType:
		return 0
	case variants.Heterozygous:
		return 1
	case variants.Homozygous:
		return 2
	case variants.Hemizygous:
		return 1
	default:
		panic("zygosity not found in genotyped cell")
		return 0
	}
}

// updateCellVar updates the CellVar slice in each Variant after removing variants during filtering
func updateCellVar(cellVars []variants.CellVar, ignoreVariants []bool, newVariantIds []int) []variants.CellVar {
	answer := make([]variants.CellVar, 0, len(cellVars))
	for _, currCv := range cellVars {
		if ignoreVariants[currCv.Vid] {
			continue
		}
		currCv.Vid = newVariantIds[currCv.Vid]
		answer = append(answer, currCv)
	}
	return answer
}

// updateCellIds updates the slice of cell ids stored in each Variant after removing cells during filtering
func updateCellIds(cellIds []int, ignoreCells []bool, newCellIds []int) []int {
	answer := make([]int, 0, len(cellIds))
	for _, oldId := range cellIds {
		if ignoreCells[oldId] {
			continue
		}
		answer = append(answer, newCellIds[oldId])
	}
	return answer
}

// fetchPassingCellsAndVariants retrieves all cells and variants that pass filters according to ignoreCells and ignoreVariants.
// The newCellIds and newVariantIds returns record the old Id for each cell/variant so that other fields can be updated later.
func fetchPassingCellsAndVariants(d *Data, ignoreCells []bool, ignoreVariants []bool) (passingCells []Cell, passingVariants []variants.Variant, newCellIds []int, newVariantIds []int) {
	passingCells = make([]Cell, int(countFalse(ignoreCells)))
	passingVariants = make([]variants.Variant, int(countFalse(ignoreVariants)))
	newCellIds = make([]int, len(d.Cells))       // such that newCellIds[oldId] is the new Id after filtering
	newVariantIds = make([]int, len(d.Variants)) // such that newVariantIds[oldId] is the new Id after filtering
	var currCellIdx, currVariantIdx int

	// fetch passing cells
	for i := range ignoreCells {
		if !ignoreCells[i] {
			passingCells[currCellIdx] = d.Cells[i]     // copy to new slice
			passingCells[currCellIdx].Id = currCellIdx // reset id
			newCellIds[i] = currCellIdx                // link old and new id for later
			currCellIdx++
		}
	}

	// fetch passing variants
	for i := range ignoreVariants {
		if !ignoreVariants[i] {
			passingVariants[currVariantIdx] = d.Variants[i]     // copy to new slice
			passingVariants[currVariantIdx].Id = currVariantIdx // reset id
			newVariantIds[i] = currVariantIdx                   // link old and new id for later
			currVariantIdx++
		}
	}
	return
}

// countFalse returns a float64 with the number of false in a []bool
func countFalse(b []bool) float64 {
	var answer float64
	for i := range b {
		if !b[i] {
			answer++
		}
	}
	return answer
}

// countPassingCellVar returns a float64 with the number of CellVar passing filters
func countPassingCellVar(v []variants.CellVar, ignoreVariants []bool) float64 {
	var answer float64
	for i := range v {
		if !ignoreVariants[v[i].Vid] {
			answer++
		}
	}
	return answer
}
