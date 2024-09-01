# go-pray

A simple cli program to print the time to the next prayer.

# Installation

```sh
go install git.sr.ht/~rehandaphedar/go-pray@latest
```

# Configuration

Edit `$XDG_CONFIG_DIR/go-pray/config.yaml`.

## City

Replace `New York` with your city's name:

```yaml
city: New York
```

## Country

Replace `USA` with your country code:

```yaml
country: USA
```

## Method

Replace `1` with your preferred method:

```yaml
method: "1"
```

Possible methods:

```json
{
	"0": "Shia Ithna-Ansari",
	"1": "University of Islamic Sciences, Karachi",
	"2": "Islamic Society of North America",
	"3": "Muslim World League",
	"4": "Umm Al-Qura University, Makkah",
	"5": "Egyptian General Authority of Survey",
	"7": "Institute of Geophysics, University of Tehran",
	"8": "Gulf Region",
	"9": "Kuwait",
	"10": "Qatar",
	"11": "Majlis Ugama Islam Singapura, Singapore",
	"12": "Union Organization islamic de France",
	"13": "Diyanet İşleri Başkanlığı, Turkey",
	"14": "Spiritual Administration of Muslims of Russia"
}
```

# Usage

```sh
go-pray
```

# Example Output

```
Asr in 01:20:25
```
