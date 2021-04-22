package cells

import (
	"github.com/vertgenlab/gonomics/dna"
	"github.com/vertgenlab/gonomics/vcf"
	"log"
	"strconv"
	"strings"
)

// BarcodeMap links each Cell to its 18bp Barcode
type BarcodeMap map[Barcode]Cell

// Cell stores information on genotypes for a single cell
type Cell struct {
	Id               int
	Genotypes        []CellVar
	GenotypesPresent float64
}

// Data organizes Cell and Variant information from a vcf file
type Data struct {
	Cells    []Cell
	Variants []Variant
}

// ReadVcf into a Data struct that stores information about Cells and Variants that pass the input filters
func ReadVcf(file string, cellFilter CellFilterParam, globalFilter GlobalFilterParam, minVcfQual float64) *Data {
	vcfChan, header := vcf.GoReadToChan(file)
	answer := new(Data)

	colNames := strings.Split(header.Text[len(header.Text)-1], "\t")
	sampleNames := colNames[9:]
	answer.Cells = make([]Cell, len(sampleNames))

	for i := range answer.Cells {
		answer.Cells[i].Id = i
	}

	for record := range vcfChan {
		if record.Qual > minVcfQual {
			parseVcf(record, cellFilter, answer)
		}
	}

	globalFilter.Apply(answer)
	return answer
}

// parseVcf to fill the appropriate fields in data
func parseVcf(v *vcf.Vcf, cellFilter CellFilterParam, data *Data) {
	var offset int
	for alleleIdx := range v.Alt { // for each allele make a new variant
		var variant Variant
		variant.Id = len(data.Variants)
		variant.Chr = v.Chr
		variant.Pos = v.Pos - 1
		variant.Ref = dna.StringToBases(v.Ref)
		variant.Alt = dna.StringToBases(v.Alt[alleleIdx])
		variant.Ref, variant.Alt, offset = trimMatchingBases(variant.Ref, variant.Alt)
		variant.Pos += offset
		variant = processCells(v, variant, alleleIdx, cellFilter, data)
		data.Variants = append(data.Variants, variant)
	}
}

// processCells parses all cells from a given vcf record and stores them directly in data
func processCells(v *vcf.Vcf, variant Variant, alleleIdx int, cellFilter CellFilterParam, data *Data) Variant {
	var currCv CellVar
	for idx := range v.Samples {
		currCv = getCellVar(v.Samples[idx], alleleIdx, variant)

		if currCv.GenotypeQuality > cellFilter.MinGenotypeQuality &&
			currCv.ReadDepth > cellFilter.MinGenotypeDepth {

			variant.CellsGenotyped = append(variant.CellsGenotyped, idx)
			if currCv.Af > cellFilter.MinReadAf {
				variant.CellsMutated = append(variant.CellsMutated, idx)
			} else if currCv.Genotype != WildType {
				currCv.Genotype = WildType
			}
		}
		data.Cells[idx].Genotypes = append(data.Cells[idx].Genotypes, currCv)
	}
	return variant
}

// getCellVar parses a GenomeSample into a CellVar
func getCellVar(g vcf.GenomeSample, alleleIdx int, variant Variant) CellVar {
	var answer CellVar
	var err error

	answer.Vid = variant.Id
	if g.AlleleOne == -1 && g.AlleleTwo == -1 {
		return answer
	}

	answer.Genotype = getZygosity(g, alleleIdx+1)
	answer.GenotypeQuality, err = strconv.Atoi(g.FormatData[3])
	if err != nil {
		answer.GenotypeQuality = 0
	}

	answer.ReadDepth, err = strconv.Atoi(g.FormatData[2])
	if err != nil {
		answer.ReadDepth = 0
	}

	altReadsPerAllele := strings.Split(g.FormatData[1], ",")
	answer.AltReads, err = strconv.Atoi(altReadsPerAllele[alleleIdx+1])
	if err != nil {
		answer.AltReads = 0
	}

	answer.Af = float64(answer.AltReads) / float64(answer.ReadDepth)
	return answer
}

// getZygosity parses a GenomeSample and returns the variant Zygosity
func getZygosity(g vcf.GenomeSample, alleleIdx int) Zygosity {
	var alleleCount int
	if g.AlleleOne == int16(alleleIdx) {
		alleleCount++
	}
	if g.AlleleTwo == -1 && alleleCount == 1 {
		return Hemizygous
	}
	if g.AlleleTwo == int16(alleleIdx) {
		alleleCount++
	}

	switch alleleCount {
	case 0:
		return WildType
	case 1:
		return Heterozygous
	case 2:
		return Homozygous
	default:
		log.Panic("could not get zygosity for", g)
		return WildType
	}
}

// trimMatchingBases removed all left-aligned matching bases in ref and alt fields.
// returns the trimmed slices and the number of bases trimmed
func trimMatchingBases(a, b []dna.Base) ([]dna.Base, []dna.Base, int) {
	var offset int
	for len(a) > 0 && len(b) > 0 {
		if a[0] == b[0] {
			a = a[1:]
			b = b[1:]
			offset++
		} else {
			break
		}
	}
	if len(a) == 0 && len(b) == 0 {
		log.Panicf("error, all bases match trimmed in\n%s\nand\n%s", dna.BasesToString(a), dna.BasesToString(b))
	}
	return a, b, offset
}
