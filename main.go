//This version can work for IPv4 and IPv6. 
//In arguments need to provide protocol (4 or 6), path to json files and name of output mmdb file. 
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"net"
	"os"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	llog "github.com/rs/zerolog/log"
)

const (
	mmdbDatabaseType = "City"
)

// GeoIPAuthRecord is a struct representing a record in the GeoIP Authoritative dataset
type GeoIPAuthRecord struct {
	Protocol int      `json:"protocol"`
	Network  string   `json:"network"`
	Location Location `json:"location"`
	//Additional Additional `json:"additional"`
}

type Location struct {
	Continent   	Continent        `json:"continent"`
	Country     	Country          `json:"country"`
	Subdivision 	Subdivision      `json:"subdivision"`
	Latitude    	Latitude         `json:"latitude"`
	Longitude   	Longitude        `json:"longitude"`
	AccuracyRadius 	AccuracyRadius 	 `json:"accuracy_radius"`
	TimeZone		string  		 `json:"time_zone"`
}

type Continent struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Country struct {
	ISOTwoDigitCode string `json:"alpha_2"`
	Name            string `json:"name"`
}

type Subdivision struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Latitude struct {
	Average           float32 `json:"average"`
	StandardDeviation float32 `json:"stddev"`
}

type Longitude struct {
	Average           float32 `json:"average"`
	StandardDeviation float32 `json:"stddev"`
}

type AccuracyRadius struct {
	Average           float32 `json:"average"`
}

type Additional struct {
	Organization string `json:"organization"`
	ISP          string `json:"isp"`
	ASN          string `json:"asn"`
	ASNName      string `json:"asn_name"`
	Company      string `json:"company"`
}

func enableIpv4Aliasing(protocol string) bool {
	if protocol == "4" {
		return false
	} else {
		return true
	}
}

func enableIpv4Version(protocol string) int {
	if protocol == "4" {
		return 4
	} else {
		return 6
	}
}

func convertAsn(asn string) (uint32, error) {
	ui64, err := strconv.ParseUint(asn, 10, 32)
	ui32 := uint32(ui64)
	return ui32, err
}

// Reads in json file and converts into a MMDB Tree
func jsonToMmdb(fileName string, writer *mmdbwriter.Tree) {
	llog.Info().Msgf("On file %s", fileName)

	linesTotal := 0
	linesSkippedEmpty := 0
	linesSkippedIPv6 := 0
	linesUnmarshalled := 0
	linesAsn := 0
	linesWritten := 0

	//path := filepath.Join(DbPath, fileName)
	rawf, err := os.Open(fileName)
	if err != nil {
		llog.Fatal().Err(err).Msg("Error opening raw file")
	}
	defer rawf.Close()

	file, err := gzip.NewReader(rawf)
	if err != nil {
		llog.Fatal().Err(err).Msgf("Unable to open gzip-compressed file")
	}

	// Create scanner for opened JSON-lines file and set line delimiter appropriately
	sc := bufio.NewScanner(file)
	sc.Split(bufio.ScanLines)

	// Go through file one line at a time, make struct and create mmdb records
	for sc.Scan() {
		linesTotal++
		b := sc.Bytes()
		// Check for empty lines and log/skip if found
		if len(b) == 0 {
			llog.Warn().Msg("Skipping empty line in JSON-lines file")
			linesSkippedEmpty++
			continue
		}

		// Unmarshal line bytes into GeoIPAuthRecord
		rec := GeoIPAuthRecord{}
		err = json.Unmarshal(sc.Bytes(), &rec)
		if err != nil {
			llog.Fatal().Err(err).Msg("Error unmarshalling bytes to GeoIPAuthRecord")
		}
		linesUnmarshalled++

		// Parse network string into net.IPNet
		// This field is required for all records in the authoritative dataset fail if a parsing issue is encountered
		_, network, err := net.ParseCIDR(rec.Network)
		if err != nil {
			llog.Fatal().Err(err).Msgf("Error parsing network %s in net.IPNet", rec.Network)
		}

		//The mmdbtype.Map is the root of the mmdb record
		// Protocol and Network aren't added into the record
		// because they are used in lookup (network, record)
		record := mmdbtype.Map{}

		record["continent"] = mmdbtype.Map{
            "code": mmdbtype.String(rec.Location.Continent.Code),
            "name": mmdbtype.String(rec.Location.Continent.Name)}
		record["country"] = mmdbtype.Map{
			"iso_code": mmdbtype.String(rec.Location.Continent.Code),
			"name": mmdbtype.String(rec.Location.Continent.Name)}
		record["location"] = mmdbtype.Map{
			"latitude": mmdbtype.Float32(rec.Location.Latitude.Average),
			"longitude": mmdbtype.Float32(rec.Location.Longitude.Average),
			"accuracy_radius": mmdbtype.Float32(rec.Location.AccuracyRadius.Average),
			"time_zone": mmdbtype.String(rec.Location.TimeZone)}
		// This defaults to inserter.ReplaceWith
		err = writer.Insert(network, record)
		if err != nil {
			llog.Fatal().Err(err).Msgf("Error inserting record %v for network %s", record, network)
		}
		linesWritten++
		if linesWritten != 0 && (linesWritten%1_000_000) == 0 {
			llog.Info().Msgf("Currently processed %d lines\n", linesWritten)
		}
	}

	llog.Info().Msgf("JSON lines in file: %d", linesTotal)
	llog.Info().Msgf("JSON lines skipped because empty: %d", linesSkippedEmpty)
	llog.Info().Msgf("IPv6 networks skipped: %d", linesSkippedIPv6)
	llog.Info().Msgf("JSON lines unmarshalled: %d", linesUnmarshalled)
	llog.Info().Msgf("JSON lines with null ASN: %d", linesAsn)
	llog.Info().Msgf("Records written to MMDB file: %d", linesWritten)

}

func main() {

	// Get protocol and file destination from arguments
	args := os.Args[1:]

	// Check if there are exactly 3 arguments provided
    if len(args) != 3 {
        fmt.Println("Usage: program <protocol> <pathToFiles> <output_db_name>")
        return
    }

	// Parse the protocol argument
    protocol := args[0]

	// Access the pathToFiles argument directly
	pathToFiles := args[1]
	pathToFiles += "*.gz"
	llog.Info().Msgf("Path to files: %s", pathToFiles)
	// This will be the mmdb file's name
	mmdbFileName := args[2]

	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            mmdbDatabaseType,
			DisableIPv4Aliasing:     enableIpv4Aliasing(protocol),
			IncludeReservedNetworks: true,
			IPVersion:               enableIpv4Version(protocol),
		},
	)
	if err != nil {
		llog.Fatal().Msgf("Creating a new mmdbwriter failed. Error: %s", err)
	}

	files, err := filepath.Glob(pathToFiles)
	if err != nil {
		llog.Fatal().Err(err)
	}

	for _, file := range files {
		llog.Info().Msgf("Working on file: %+v", file)
		//Create mmdb file
		jsonToMmdb(file, writer)
		llog.Info().Msgf("Current state of MMDB writer: %+v", writer)
	}

	fh, err := os.Create(mmdbFileName)
	if err != nil {
		llog.Fatal().Msgf("%s file failed to be created: %s", mmdbFileName, err)
	}

	_, err = writer.WriteTo(fh)
	if err != nil {
		llog.Fatal().Msgf("%s file was failed to be written to: %s", mmdbFileName, err)
	}

	fh.Close()
}
