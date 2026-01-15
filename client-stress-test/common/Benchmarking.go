package common

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"os"
	"sort"
	"strings"
	"time"
)

func GetHeader() []string {
	return []string{
		"name",
		"created_time",
		"users_clients",
		"users_connected",
		"users_joined",
		"subscriptions_sent",
		"subscriptions_received",
		"cpu",
		"mem",
		"connection_established",
		"connection_ack_received",
		"join_start",
		"join_sent",
		"join_completed",
		"join_received",
		"user_current_data_received",
		"chat_message_completed",
		"left",
		"left_time",
	}
}

func WriteHeaderToCsv(writer *csv.Writer) {
	if err := writer.Write(GetHeader()); err != nil {
		fmt.Println("Cannot write headers to CSV", err)
		return
	}
	writer.Flush()
}

func WriteToCsv(writer *csv.Writer, user *User) {
	row := make([]string, len(GetHeader()))
	for i, header := range GetHeader() {
		value := user.BenchmarkingMetrics[header]
		switch v := value.(type) {
		case string:
			row[i] = v
		case int, int32, int64, float32, float64:
			row[i] = fmt.Sprintf("%v", v)
		default:
			// Use json.Marshal for complex types
			if jsonBytes, err := json.Marshal(v); err == nil {
				row[i] = string(jsonBytes)
			} else {
				row[i] = ""
			}
		}
	}
	if err := writer.Write(row); err != nil {
		user.Logger.Fatal("Cannot write row to CSV", err)
		//return
	}

	writer.Flush()
}

var users []*User

func AddBenckmarkingUser(user *User) {
	users = append(users, user)
}

func SortUsers() {
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})
}

func ExportCsv(fileName string) {
	SortUsers()

	file, _ := os.Create("benchmarking/" + fileName + ".csv")
	writer := csv.NewWriter(file)
	defer writer.Flush()

	WriteHeaderToCsv(writer)
	for _, user := range users {
		WriteToCsv(writer, user)
	}
}

func DrawPlot(fileName string) {
	if len(users) == 0 {
		fmt.Printf("No benchmark users found to generate a chart.\n")
		return
	}

	fmt.Printf("Generation chart for %d users.\n", len(users))

	//Sort users
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	p := plot.New()

	title := "Graphql Stress Test"
	title += fmt.Sprintf(" (%d users)\n", GetConfig().NumOfUsers)
	if GetConfig().SendSubscriptionsBatch {
		title += "Subscriptions Batch: Enabled\n"
	} else {
		title += "Subscriptions Batch: Disabled\n"
	}
	if GetConfig().MinIntervalBetweenUserJoinInMs != GetConfig().MaxIntervalBetweenUserJoinInMs {
		title += fmt.Sprintf("Interval between joins: [%d - %d]ms\n", GetConfig().MinIntervalBetweenUserJoinInMs, GetConfig().MaxIntervalBetweenUserJoinInMs)
	} else {
		title += fmt.Sprintf("Interval between joins: %dms\n", GetConfig().MinIntervalBetweenUserJoinInMs)
	}

	p.Title.Text = title
	w := vg.Points(20)

	metrics := []string{"connection_established", "connection_ack_received", "join_sent", "join_completed", "join_received", "connection_alive_completed", "chat_message_completed"}
	for i := len(metrics) - 1; i >= 0; i-- {
		var metricTimeValues []float64
		for _, user := range users {
			currDuration, exists := user.BenchmarkingMetrics[metrics[i]].(time.Duration)

			if exists {
				metricTimeValues = append(metricTimeValues, float64(currDuration.Milliseconds()))
			} else {
				metricTimeValues = append(metricTimeValues, 0)
			}
		}

		metricTimes := plotter.Values(metricTimeValues)

		bars, err := plotter.NewBarChart(metricTimes, w)
		if err != nil {
			panic(err)
		}
		//barsA.LineStyle.Width = vg.Length(0)
		bars.Color = plotutil.Color(i)
		//barsA.Offset = -w
		//barsA.Offset = vg.Points(float64(0) * (20 + 2))
		bars.Offset = 0 //All with same offset (stacked)
		p.Add(bars)
		p.Legend.Add(metrics[i], bars)
	}

	overviewLines := []string{"users_connected", "users_joined"}
	for j, overviewLine := range overviewLines {
		var lineValues []float64
		for _, user := range users {
			userLineValue, exists := user.BenchmarkingMetrics[overviewLine].(int)
			if exists {
				lineValues = append(lineValues, float64(userLineValue)*5)
			} else {
				lineValues = append(lineValues, 0)
			}
		}

		//Add Line
		pts := make(plotter.XYs, len(lineValues))
		for i, curr := range lineValues {
			pts[i].X = float64(i)
			pts[i].Y = curr
		}
		lineData := plotter.XYs(pts)

		line, err := plotter.NewLine(lineData)
		if err != nil {
			panic(err)
		}

		if overviewLine == "users_connected" {
			line.Color = plotutil.Color(0)
		} else if overviewLine == "users_joined" {
			line.Color = plotutil.Color(2)
		} else {
			line.Color = plotutil.Color(j)
		}

		line.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
		line.Width = vg.Points(2)

		p.Add(line)
		p.Legend.Add(overviewLine, line)

	}

	p.Y.Label.Text = "Elapsed time (ms)"
	p.Y.Min = 0
	p.Y.Max = 50000
	p.Y.Tick.Marker = commaTicks{}
	//p.Y.Tick.Marker = plot.DefaultTicks{} // Use default ticks for left Y-axis
	//rightAxis := plotter.NewYAxis("Right Y-axis")
	//rightAxis.Scale = plot.LinearScale{}        // Use a linear scale or customize as needed
	//rightAxis.Tick.Marker = plot.DefaultTicks{} // Customize ticks for the right Y-axis
	//p.Add(rightAxis)

	//var userCurrentTimeValues []float64
	//var joinedUsersValues []float64
	//var ackUsersValues []float64
	//var connectedUsersValues []float64
	var names []string
	for _, user := range users {
		connectedUsers := user.BenchmarkingMetrics["users_connected"].(int)
		//connectedUsersValues = append(connectedUsersValues, float64(connectedUsers))
		//
		//ackReceivedUsers := user.BenchmarkingMetrics["connection_ack_received"].(int)
		//ackUsersValues = append(ackUsersValues, float64(ackReceivedUsers))
		//
		joinedUsers := user.BenchmarkingMetrics["users_joined"].(int)
		//joinedUsersValues = append(joinedUsersValues, float64(joinedUsers))

		subscriptionsSent := user.BenchmarkingMetrics["subscriptions_sent"].(int)
		subscriptionsReceived := user.BenchmarkingMetrics["subscriptions_received"].(int)
		cpu := user.BenchmarkingMetrics["cpu"].(string)

		//connectedUsers := user.BenchmarkingMetrics["users_connected"].(int)
		name := fmt.Sprintf("%s\nUsers: (%d/%d)\nSubs: (%s/%s)\nCPU usage: %s", user.Name, connectedUsers, joinedUsers, formatFloat(float64(subscriptionsSent), 0), formatFloat(float64(subscriptionsReceived), 0), cpu)
		names = append(names, name)
	}

	p.Legend.Top = true
	p.NominalX(names...)

	if err := p.Save(vg.Length(len(users)+1)*vg.Points(100), 15*vg.Inch, "benchmarking/"+fileName+".png"); err != nil {
		panic(err)
	}
}

type commaTicks struct{}

func (commaTicks) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for val := min; val <= max; val += 500 {
		ticks = append(ticks, plot.Tick{Value: val, Label: formatFloat(val, 0)})
	}
	return ticks
}

func insertCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var buffer bytes.Buffer
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 {
			buffer.WriteRune(',')
		}
		buffer.WriteRune(r)
	}
	return buffer.String()
}

func formatFloat(f float64, decimalPlaces int) string {
	str := fmt.Sprintf("%.*f", decimalPlaces, f)
	parts := strings.Split(str, ".")
	parts[0] = insertCommas(parts[0])
	return strings.Join(parts, ".")
}
