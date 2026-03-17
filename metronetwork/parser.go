package metronetwork

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"github.com/sirupsen/logrus"
	"github.com/xorcl/api-red/common"
)

const KeyValURL = "https://www.metro.cl/api/estadoRedDetalle.php"
const TimeURL = "https://www.metro.cl/api/horariosEstacion.php?cod=%s"
const HolidayURL = "https://apis.digital.gob.cl/fl/feriados/%d/%d/%d"

var slugRegex = regexp.MustCompile(`-l\da?$`)

type Parser struct {
	StationTimes map[string]*CompositeTime
	IsHoliday    bool
}

func (bp *Parser) GetRoute() string {
	return "metro-network"
}

func (bp *Parser) StartParser() {
}

func (bp *Parser) Parse(c *gin.Context) {
	response := &Response{
		Lines: make([]*LineResponse, 0),
		Time:  time.Now().Format("2006-01-02 15:04:05"),
	}

	resp, err := http.Get(KeyValURL)
	if err != nil {
		logrus.Errorf("Error retrieving Metro Status: %s", err)
		response.APIStatus = "Error al conectarse al sitio de Metro"
		c.JSON(400, &response)
		return
	}
	defer resp.Body.Close()

	kv := make(KeyValResponse)
	err = json.NewDecoder(resp.Body).Decode(&kv)
	if err != nil {
		logrus.Errorf("Error parsing Metro Status: %s", err)
		response.APIStatus = "Error al procesar datos de Metro"
		c.JSON(400, &response)
		return
	}

	// Sort line keys for consistent ordering (l1, l2, l3, l4, l4a, l5, l6)
	lineKeys := make([]string, 0, len(kv))
	for k := range kv {
		lineKeys = append(lineKeys, k)
	}
	sort.Strings(lineKeys)

	transferStations := make(map[string][]string)

	for _, key := range lineKeys {
		lineData := kv[key]
		lineID := strings.ToUpper(key)
		lineNum := strings.TrimPrefix(lineID, "L")

		lineStatus := stringToStatusCode[lineData.Estado]
		line := &LineResponse{
			Name:     "Línea " + lineNum,
			ID:       lineID,
			Issues:   lineStatus != 0,
			Stations: make([]*StationResponse, 0),
		}
		if line.Issues {
			response.Issues = true
		}

		for _, stData := range lineData.Estaciones {
			stStatus := stringToStatusCode[stData.Estado]
			if stStatus != 0 {
				line.Issues = true
				response.Issues = true
			}

			// Strip line suffix from display name ("San Pablo L1" -> "San Pablo")
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(stData.Nombre), " "+lineID))
			stSlug := slugRegex.ReplaceAllString(slug.Make(strings.TrimSpace(stData.Nombre)), "")

			// Track which lines each station belongs to
			transferStations[name] = append(transferStations[name], lineID)
			if stData.Combinacion != "" {
				comboID := strings.ToUpper(strings.TrimSpace(stData.Combinacion))
				transferStations[name] = append(transferStations[name], comboID)
			}

			station := &StationResponse{
				Name:        name,
				ID:          stSlug,
				Status:      stStatus,
				Description: strings.TrimSpace(stData.Descripcion),
				Reason:      strings.TrimSpace(stData.Mensaje),
			}

			if ct, ok := bp.StationTimes[stSlug]; ok {
				station.Schedule = ct
				if closed, err := ct.IsClosed(bp.IsHoliday); closed && err == nil {
					station.IsClosedBySchedule = true
					line.StationsClosedBySchedule++
				}
			}

			line.Stations = append(line.Stations, station)
		}

		response.Lines = append(response.Lines, line)
	}

	// Set deduplicated transfer lines for each station
	for _, line := range response.Lines {
		for _, station := range line.Stations {
			if lines, ok := transferStations[station.Name]; ok {
				seen := make(map[string]bool)
				unique := make([]string, 0)
				for _, l := range lines {
					if !seen[l] {
						seen[l] = true
						unique = append(unique, l)
					}
				}
				station.Lines = unique
			}
		}
	}

	response.APIStatus = "OK"
	c.JSON(200, &response)
}

func (bp *Parser) StopParser() {
}

func (p *Parser) GetCronTasks() []*common.CronTask {
	return []*common.CronTask{
		{
			Name: "Get all stations",
			Time: "0 1 * * *",
			Execute: func() error {
				logrus.Infof("checking station schedules...")
				kv := make(KeyValResponse)
				completeNames := make(map[string]string)
				resp, err := http.Get(KeyValURL)
				if err != nil {
					logrus.Errorf("Error retrieving Metro Status page: %s", err)
					return err
				}
				defer resp.Body.Close()
				err = json.NewDecoder(resp.Body).Decode(&kv)
				if err != nil {
					logrus.Errorf("Error parsing Metro Status body: %s", err)
					return err
				}
				for _, v := range kv {
					for _, station := range v.Estaciones {
						stSlug := strings.TrimSpace(slug.Make(station.Nombre))
						completeNames[station.Codigo] = slugRegex.ReplaceAllString(stSlug, "")
					}
				}
				p.StationTimes = make(map[string]*CompositeTime)
				for code, name := range completeNames {
					logrus.Infof("checking schedule for %s...", name)
					p.StationTimes[name] = &CompositeTime{}
					sr := ScheduleResponse{}
					resp, err := http.Get(fmt.Sprintf(TimeURL, code))
					if err != nil {
						logrus.Errorf("Error retrieving %s station schedule page: %s", name, err)
						continue
					}
					err = json.NewDecoder(resp.Body).Decode(&sr)
					resp.Body.Close()
					if err != nil {
						logrus.Errorf("Error parsing Metro Status body: %s", err)
						continue
					}
					p.StationTimes[name].Open.Weekdays = strings.TrimSpace(sr.Estacion.Abrir.LunesViernes)
					p.StationTimes[name].Open.Saturday = strings.TrimSpace(sr.Estacion.Abrir.Sabado)
					p.StationTimes[name].Open.Holidays = strings.TrimSpace(sr.Estacion.Abrir.Domingo)

					p.StationTimes[name].Close.Weekdays = strings.TrimSpace(sr.Estacion.Cerrar.LunesViernes)
					p.StationTimes[name].Close.Saturday = strings.TrimSpace(sr.Estacion.Cerrar.Sabado)
					p.StationTimes[name].Close.Holidays = strings.TrimSpace(sr.Estacion.Cerrar.Domingo)
				}
				return nil
			},
		},
		{
			Name: "Is holiday today?",
			Time: "0 0 * * *",
			Execute: func() error {
				logrus.Infof("checking if today is holiday...")
				p.IsHoliday = false
				holidays := make([]struct{}, 0)
				now := time.Now()
				resp, err := http.Get(fmt.Sprintf(HolidayURL, now.Year(), now.Month(), now.Day()))
				if err != nil {
					logrus.Errorf("Error retrieving Gob Holiday API: %s", err)
					return err
				}
				defer resp.Body.Close()
				err = json.NewDecoder(resp.Body).Decode(&holidays)
				if err == nil && len(holidays) > 0 {
					p.IsHoliday = true
				}
				logrus.Infof("today is holiday: %t", p.IsHoliday)
				return nil
			},
		},
	}
}
