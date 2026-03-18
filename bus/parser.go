package bus

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xorcl/api-red/common"
)

const BASE_URL = "https://www.red.cl/predictorPlus/prediccion?t=%s&codsimt=%s&codser="
const SESSION_URL = "https://www.red.cl/planifica-tu-viaje/cuando-llega/"

var (
	jwtRegexp  = regexp.MustCompile(`\$jwt = '([A-Za-z0-9+/=_-]+)'`)
	regexMenos = regexp.MustCompile(`(?i)en menos de (\d+)`)
	regexEntre = regexp.MustCompile(`(?i)entre (\d+) y (\d+)`)
	regexMas   = regexp.MustCompile(`(?i)mas de (\d+)`)
)

type Parser struct {
	Session string
}

func (bp *Parser) GetRoute() string {
	return "bus-stop/:stopid"
}

func (bp *Parser) StartParser() {
}

func (bp *Parser) Parse(c *gin.Context) {
	bp.getSession()
	stopID := c.Param("stopid")
	url := fmt.Sprintf(BASE_URL, bp.Session, stopID)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching predictor for bus-stop parser: %s", err)
		c.JSON(500, gin.H{"error": "No puedo obtener la información"})
		return
	}
	defer resp.Body.Close()

	var predictor PredictorResponse
	if err := json.NewDecoder(resp.Body).Decode(&predictor); err != nil {
		log.Printf("Error decoding predictor response: %s", err)
		c.JSON(500, gin.H{"error": "Error procesando respuesta"})
		return
	}

	response := BusStopResponse{
		ID:       predictor.Paradero,
		Name:     predictor.Nomett,
		Services: make([]Service, 0),
	}

	for _, item := range predictor.GetServices() {
		service := Service{
			ID:                item.Servicio,
			Valid:             true,
			StatusDescription: item.RespuestaServicio,
			Buses:             make([]Bus, 0),
		}
		if b, ok := parseBus(item.PpuBus1, item.DistanciaBus1, item.HoraPrediccionBus1); ok {
			service.Buses = append(service.Buses, b)
		}
		if b, ok := parseBus(item.PpuBus2, item.DistanciaBus2, item.HoraPrediccionBus2); ok {
			service.Buses = append(service.Buses, b)
		}
		response.Services = append(response.Services, service)
	}

	c.JSON(200, response)
}

func (bp *Parser) StopParser() {
}

func (bp *Parser) getSession() {
	resp, err := http.Get(SESSION_URL)
	if err != nil {
		log.Printf("Error getting session for bus-stop parser: %s", err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Printf("Error reading session page for bus-stop parser: %s", err)
		return
	}
	jwtB64 := jwtRegexp.FindSubmatch(body)
	if jwtB64 == nil {
		log.Printf("Error getting session for bus-stop parser: JWT not found in page")
		return
	}
	jwt, err := base64.StdEncoding.DecodeString(string(jwtB64[1]))
	if err != nil {
		log.Printf("Error decoding jwt for bus-stop parser: %s", err)
		return
	}
	bp.Session = string(jwt)
}

func (p *Parser) GetCronTasks() []*common.CronTask {
	return make([]*common.CronTask, 0)
}

func parseBus(ppu, distancia, horario string) (Bus, bool) {
	if ppu == "" || horario == "" {
		return Bus{}, false
	}
	min, max, ok := parseArrivalTime(horario)
	if !ok {
		return Bus{}, false
	}
	dist, _ := strconv.Atoi(distancia)
	return Bus{
		ID:             ppu,
		MetersDistance: dist,
		MinArrivalTime: min,
		MaxArrivalTime: max,
	}, true
}

func parseArrivalTime(s string) (min, max int, ok bool) {
	if m := regexMenos.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return 0, n, true
	}
	if m := regexEntre.FindStringSubmatch(s); m != nil {
		n1, _ := strconv.Atoi(m[1])
		n2, _ := strconv.Atoi(m[2])
		return n1, n2, true
	}
	if m := regexMas.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, 99, true
	}
	return 0, 0, false
}
