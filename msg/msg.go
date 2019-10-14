// Package msg provides means for client and server to communicate.
package msg

type QueryParam []string

type Cmd struct {
	Op          string            `json:"operation"`    // The operation to perform
	Flags       map[string]bool   `json:"flags"`        // Possible flags
	Opts        map[string]string `json:"options"`      // Possible options
	Tasks       []string          `json:"tasks"`        // The tasks for any related requests
	Body        [][]string        `json:"body"`         // The body containing the command information
	QueryParams []QueryParam      `json:"query_params"` // The parameters for a query
}
