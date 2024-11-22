package banner

import (
	"fmt"
)

// prints the version message
const version = "v0.0.4"

func PrintVersion() {
	fmt.Printf("Current linkinspector version %s\n", version)
}

// Prints the Colorful banner
func PrintBanner() {
	banner := `
    __ _         __    _                                  __              
   / /(_)____   / /__ (_)____   _____ ____   ___   _____ / /_ ____   _____
  / // // __ \ / //_// // __ \ / ___// __ \ / _ \ / ___// __// __ \ / ___/
 / // // / / // ,<  / // / / /(__  )/ /_/ //  __// /__ / /_ / /_/ // /    
/_//_//_/ /_//_/|_|/_//_/ /_//____// .___/ \___/ \___/ \__/ \____//_/     
                                  /_/
`
	fmt.Printf("%s\n%75s\n\n", banner, "Current linkinspector version "+version)
}
