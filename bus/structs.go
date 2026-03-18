package bus

import "encoding/json"

// BusStopResponse is the format returned to the app
type BusStopResponse struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Services []Service `json:"services"`
}

type Service struct {
	ID                string `json:"id"`
	Valid             bool   `json:"valid"`
	StatusDescription string `json:"status_description"`
	Buses             []Bus  `json:"buses"`
}

type Bus struct {
	ID             string `json:"id"`
	MetersDistance int    `json:"meters_distance"`
	MinArrivalTime int    `json:"min_arrival_time"`
	MaxArrivalTime int    `json:"max_arrival_time"`
}

// PredictorResponse is the raw format from red.cl predictorPlus
type PredictorResponse struct {
	Nomett    string `json:"nomett"`
	Paradero  string `json:"paradero"`
	Servicios struct {
		Item json.RawMessage `json:"item"`
	} `json:"servicios"`
}

// GetServices handles the case where red.cl returns either an array or a single object
func (p *PredictorResponse) GetServices() []PredictorService {
	var services []PredictorService
	if err := json.Unmarshal(p.Servicios.Item, &services); err == nil {
		return services
	}
	var single PredictorService
	if err := json.Unmarshal(p.Servicios.Item, &single); err == nil {
		return []PredictorService{single}
	}
	return nil
}

type PredictorService struct {
	Servicio           string `json:"servicio"`
	PpuBus1            string `json:"ppubus1"`
	PpuBus2            string `json:"ppubus2"`
	DistanciaBus1      string `json:"distanciabus1"`
	DistanciaBus2      string `json:"distanciabus2"`
	HoraPrediccionBus1 string `json:"horaprediccionbus1"`
	HoraPrediccionBus2 string `json:"horaprediccionbus2"`
	RespuestaServicio  string `json:"respuestaServicio"`
}
