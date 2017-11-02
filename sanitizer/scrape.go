package sanitizer

import (
	"errors"
	"fmt"
	"github.com/iBatStat/extractor/model"
	gotes "github.com/otiai10/gosseract"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ocr *gotes.Client
)

func init() {
	var err error
	ocr, err = gotes.NewClient()
	if err != nil {
		log.Fatal(err)
		return
	}
}

type DurType string

const (
	HR  DurType = "hr"
	MIN DurType = "min"
)

var (
	usgRe        = regexp.MustCompile(`(Usage\s+.*\n)`)
	standbyRe    = regexp.MustCompile(`(Standby\s+.*\n)`)
	batPercentRe = regexp.MustCompile(`((AM|PM).*\d%)`)
	statTimeRe   = regexp.MustCompile(`(.*(\s)(AM|PM))`)
)

func explodeTypeAndVal(strDur string) (DurType, int, error) {
	var durType DurType
	var val int
	var err error
	if strings.Contains(strDur, string(HR)) {
		durType = HR
		val, err = strconv.Atoi(strings.TrimSpace(strings.Split(strDur, string(HR))[0]))
	} else {
		if strings.Contains(strDur, string(MIN)) {
			val, err = strconv.Atoi(strings.TrimSpace(strings.Split(strDur, string(MIN))[0]))
			durType = MIN
		} else {
			return "", -1, errors.New(fmt.Sprintf("Data (%s) has invalid time units", strDur))
		}
	}
	return durType, val, err
}
func extractTimes(timesArr []string) (time.Duration, error) {
	var durData time.Duration
	for _, data := range timesArr {
		t, v, err := explodeTypeAndVal(data)
		if err != nil {
			return -1 * time.Hour, err
		} else {
			switch t {
			case HR:
				durData = durData + time.Duration(v)*time.Hour
			case MIN:
				durData = durData + time.Duration(v)*time.Minute
			}
		}
	}
	return durData, nil
}
func extractBatteryTimes(exp string) (time.Duration, time.Duration, error) {
	var usgStr, standbyStr, batPercentStr, statTimeStr string
	for _, match := range usgRe.FindAllString(exp, -1) {
		usgStr = strings.TrimSpace(strings.Split(match, "Usage")[1])
	}
	for _, match := range standbyRe.FindAllString(exp, -1) {
		standbyStr = strings.TrimSpace(strings.Split(match, "Standby")[1])
	}
	for _, match := range batPercentRe.FindAllString(exp, -1) {
		filter := strings.Split(match, " ")
		batPercentStr = strings.TrimSpace(filter[len(filter)-1])
	}
	for _, match := range statTimeRe.FindAllString(exp, -1) {
		fmt.Println(fmt.Sprintf("Stat collection rawtime is (%v)", match))
		filter := strings.Split(match, " ")
		statTimeStr = strings.TrimSpace(filter[len(filter)-1])
	}
	fmt.Println(fmt.Sprintf("Usage rawtimes are (%v)", usgStr))
	fmt.Println(fmt.Sprintf("Standby rawtimes are (%v)", standbyStr))
	fmt.Println(fmt.Sprintf("Battery percentage is (%v)", batPercentStr))
	fmt.Println(fmt.Sprintf("Stat collection time is (%v)", statTimeStr))
	usgdata := strings.Split(usgStr, ",")
	standbydata := strings.Split(standbyStr, ",")
	usgdur, err := extractTimes(usgdata)
	standbydur, err := extractTimes(standbydata)
	return usgdur, standbydur, err
}

func ExtractFeatures(imgPath string) (*model.BatteryStats, error) {
	out := gotes.Must(gotes.Params{
		Src:       imgPath,
		Languages: "eng",
	})
	//out, err := ocr.Src(imgPath).Out()
	//if err != nil {
	//		log.Fatal(err)
	//		return nil, err
	//	}
	fmt.Println(fmt.Sprintf("****** Raw data is *********\n%s", out))
	//TO DO Use ocr to extract data nd the construct the BatterStats out of it
	usgdata, standbydur, err := extractBatteryTimes(out)
	batStat := model.BatteryStats{}
	if err != nil {
		return nil, err
	} else {
		batStat.Usage = usgdata
		batStat.Standby = standbydur
		return &batStat, nil
	}
}
