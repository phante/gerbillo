package main

import "testing"

func TestInterpretaPunto(t *testing.T) {
	for in, out := range map[string]string{
		"46.2905, 8.2987":             "46.2905, 8.2987",
		"46.2905 8.2987":              "46.2905, 8.2987",
		"46,2905; 8,2987":             "46.2905, 8.2987",
		"-33.8688, 151.2093":          "-33.8688, 151.2093",
		`46°17'26"N 8°17'55"E`:        "46.290556, 8.298611",
		`46° 17′ 26″ N, 8° 17′ 55″ E`: "46.290556, 8.298611",
		`N 46°17.433' E 8°17.917'`:    "46.29055, 8.298617",
		`33°52'08"S 151°12'33"E`:      "-33.868889, 151.209167",
		`8°17'55"E 46°17'26"N`:        "46.290556, 8.298611",
		"https://www.google.com/maps/place/Diga/@46.2901,8.2950,15z/data=!3m1!4b1!4m6!3m5!1s0x0:0x0!8m2!3d46.2905!4d8.2987": "46.2905, 8.2987",
		"https://www.google.com/maps/@46.2901,8.295,15z":                                                                    "46.2901, 8.295",
		"https://maps.google.com/?q=46.2905,8.2987":                                                                         "46.2905, 8.2987",
		"https://www.google.com/maps/search/?api=1&query=46.2905%2C8.2987":                                                  "46.2905, 8.2987",
	} {
		lat, lon, err := interpretaPunto(in)
		if err != nil || testoPunto(lat, lon) != out {
			t.Errorf("%q: %s, %v; atteso %s", in, testoPunto(lat, lon), err, out)
		}
	}
	for _, in := range []string{"46,2672", "46,2905, 8,2987", "Pippo", "95, 10", "46.29", "https://example.com/x"} {
		if lat, lon, err := interpretaPunto(in); err == nil {
			t.Errorf("%q accettato: %v, %v", in, lat, lon)
		}
	}
}
