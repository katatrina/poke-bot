package crawler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

type PokemonDBCrawler struct {
	collector *colly.Collector
	baseURL   string
}

func NewPokemonDBCrawler() *PokemonDBCrawler {
	c := colly.NewCollector(
		colly.AllowedDomains("pokemondb.net"),
		colly.MaxDepth(2),
		colly.Async(false), // Synchronous for controlled crawling
	)
	
	// Set delays to be respectful
	c.Limit(&colly.LimitRule{
		DomainGlob:  "pokemondb.net",
		Delay:       500 * time.Millisecond,
		RandomDelay: 200 * time.Millisecond,
	})
	
	// Use random user agent
	extensions.RandomUserAgent(c)
	
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error crawling %s: %v", r.Request.URL, err)
	})
	
	return &PokemonDBCrawler{
		collector: c,
		baseURL:   "https://pokemondb.net",
	}
}

type PokemonData struct {
	Name          string
	Number        string
	Types         []string
	Stats         map[string]int
	Abilities     []string
	Description   string
	Height        string
	Weight        string
	Category      string
	Evolutions    []string
	WeakAgainst   []string
	StrongAgainst []string
	Generation    int
}

func (pc *PokemonDBCrawler) CrawlPokemonList(ctx context.Context, limit int) ([]string, error) {
	var pokemonURLs []string
	count := 0
	
	pc.collector.OnHTML("div.infocard-list-pkmn-lg > div.infocard", func(e *colly.HTMLElement) {
		if count >= limit {
			return
		}
		
		// Get Pokemon URL
		link := e.ChildAttr("span.infocard-lg-img a", "href")
		if link != "" {
			pokemonURLs = append(pokemonURLs, pc.baseURL+link)
			count++
		}
	})
	
	// Start from National Pokedex
	err := pc.collector.Visit(pc.baseURL + "/pokedex/national")
	if err != nil {
		return nil, fmt.Errorf("failed to visit pokedex: %w", err)
	}
	
	pc.collector.Wait()
	
	return pokemonURLs, nil
}

func (pc *PokemonDBCrawler) CrawlPokemonDetails(ctx context.Context, url string) (*PokemonData, error) {
	pokemon := &PokemonData{
		Stats:         make(map[string]int),
		Types:         []string{},
		Abilities:     []string{},
		Evolutions:    []string{},
		WeakAgainst:   []string{},
		StrongAgainst: []string{},
	}
	
	detailCollector := pc.collector.Clone()
	
	// Get Pokemon name and number
	detailCollector.OnHTML("main h1", func(e *colly.HTMLElement) {
		pokemon.Name = strings.TrimSpace(e.Text)
	})
	
	// Get Pokemon number from breadcrumb or table
	detailCollector.OnHTML("table.vitals-table tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
			header := strings.TrimSpace(row.ChildText("th"))
			value := strings.TrimSpace(row.ChildText("td"))
			
			switch header {
			case "National №":
				pokemon.Number = value
			case "Type":
				row.ForEach("td a.type-icon", func(_ int, typeElem *colly.HTMLElement) {
					pokemonType := strings.TrimSpace(typeElem.Text)
					if pokemonType != "" {
						pokemon.Types = append(pokemon.Types, pokemonType)
					}
				})
			case "Species":
				pokemon.Category = value
			case "Height":
				pokemon.Height = value
			case "Weight":
				pokemon.Weight = value
			case "Abilities":
				row.ForEach("td a", func(_ int, ability *colly.HTMLElement) {
					abilityName := strings.TrimSpace(ability.Text)
					if abilityName != "" && !strings.Contains(abilityName, "(hidden ability)") {
						pokemon.Abilities = append(pokemon.Abilities, abilityName)
					}
				})
			}
		})
	})
	
	// Get base stats
	detailCollector.OnHTML("div.resp-scroll", func(e *colly.HTMLElement) {
		e.ForEach("table.vitals-table tbody tr", func(_ int, row *colly.HTMLElement) {
			statName := strings.TrimSpace(row.ChildText("th"))
			statValue := strings.TrimSpace(row.ChildText("td.cell-num"))
			
			if statValue != "" {
				// Try to parse stat value
				var value int
				fmt.Sscanf(statValue, "%d", &value)
				
				switch statName {
				case "HP":
					pokemon.Stats["HP"] = value
				case "Attack":
					pokemon.Stats["Attack"] = value
				case "Defense":
					pokemon.Stats["Defense"] = value
				case "Sp. Atk":
					pokemon.Stats["SpAttack"] = value
				case "Sp. Def":
					pokemon.Stats["SpDefense"] = value
				case "Speed":
					pokemon.Stats["Speed"] = value
				case "Total":
					pokemon.Stats["Total"] = value
				}
			}
		})
	})
	
	// Get Pokedex description
	detailCollector.OnHTML("div.grid-col:has(h2:contains('Pokédex entries')) table tbody", func(e *colly.HTMLElement) {
		// Get first available description
		e.ForEach("tr", func(i int, row *colly.HTMLElement) {
			if i == 0 && pokemon.Description == "" {
				desc := strings.TrimSpace(row.ChildText("td.cell-med-text"))
				if desc != "" {
					pokemon.Description = desc
				}
			}
		})
	})
	
	// Get type effectiveness
	detailCollector.OnHTML("div.grid-col:has(h2:contains('Type defenses'))", func(e *colly.HTMLElement) {
		e.ForEach("table.type-table tbody tr", func(_ int, row *colly.HTMLElement) {
			header := strings.TrimSpace(row.ChildText("th"))
			
			if strings.Contains(header, "damaged normally by") {
				// Skip normal damage
				return
			}
			
			row.ForEach("td a.type-icon", func(_ int, typeElem *colly.HTMLElement) {
				typeName := strings.TrimSpace(typeElem.Text)
				if typeName == "" {
					return
				}
				
				if strings.Contains(header, "weak to") || strings.Contains(header, "damaged by") {
					// Check for 2× or 4× weakness
					title := typeElem.Attr("title")
					if strings.Contains(title, "2×") || strings.Contains(title, "4×") {
						pokemon.WeakAgainst = append(pokemon.WeakAgainst, typeName)
					}
				} else if strings.Contains(header, "resistant to") || strings.Contains(header, "not very effective") {
					pokemon.StrongAgainst = append(pokemon.StrongAgainst, typeName)
				}
			})
		})
	})
	
	// Get evolution chain
	detailCollector.OnHTML("div.infocard-list-evo", func(e *colly.HTMLElement) {
		e.ForEach("div.infocard", func(_ int, evo *colly.HTMLElement) {
			evoName := strings.TrimSpace(evo.ChildText("a.ent-name"))
			if evoName != "" && evoName != pokemon.Name {
				pokemon.Evolutions = append(pokemon.Evolutions, evoName)
			}
		})
	})
	
	// Visit the Pokemon detail page
	err := detailCollector.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit pokemon page %s: %w", url, err)
	}
	
	detailCollector.Wait()
	
	// Validate we got essential data
	if pokemon.Name == "" {
		return nil, fmt.Errorf("failed to extract pokemon data from %s", url)
	}
	
	return pokemon, nil
}

func (pc *PokemonDBCrawler) FormatPokemonForRAG(pokemon *PokemonData) string {
	var sb strings.Builder
	
	// Header
	sb.WriteString(fmt.Sprintf("Pokemon: %s", pokemon.Name))
	if pokemon.Number != "" {
		sb.WriteString(fmt.Sprintf(" (#%s)", pokemon.Number))
	}
	sb.WriteString("\n\n")
	
	// Basic Info
	sb.WriteString("=== Basic Information ===\n")
	if len(pokemon.Types) > 0 {
		sb.WriteString(fmt.Sprintf("Type: %s\n", strings.Join(pokemon.Types, ", ")))
	}
	if pokemon.Category != "" {
		sb.WriteString(fmt.Sprintf("Category: %s\n", pokemon.Category))
	}
	if pokemon.Height != "" {
		sb.WriteString(fmt.Sprintf("Height: %s\n", pokemon.Height))
	}
	if pokemon.Weight != "" {
		sb.WriteString(fmt.Sprintf("Weight: %s\n", pokemon.Weight))
	}
	sb.WriteString("\n")
	
	// Description
	if pokemon.Description != "" {
		sb.WriteString("=== Description ===\n")
		sb.WriteString(pokemon.Description)
		sb.WriteString("\n\n")
	}
	
	// Abilities
	if len(pokemon.Abilities) > 0 {
		sb.WriteString("=== Abilities ===\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", strings.Join(pokemon.Abilities, ", ")))
	}
	
	// Base Stats
	if len(pokemon.Stats) > 0 {
		sb.WriteString("=== Base Stats ===\n")
		if hp, ok := pokemon.Stats["HP"]; ok {
			sb.WriteString(fmt.Sprintf("HP: %d\n", hp))
		}
		if attack, ok := pokemon.Stats["Attack"]; ok {
			sb.WriteString(fmt.Sprintf("Attack: %d\n", attack))
		}
		if defense, ok := pokemon.Stats["Defense"]; ok {
			sb.WriteString(fmt.Sprintf("Defense: %d\n", defense))
		}
		if spAttack, ok := pokemon.Stats["SpAttack"]; ok {
			sb.WriteString(fmt.Sprintf("Special Attack: %d\n", spAttack))
		}
		if spDefense, ok := pokemon.Stats["SpDefense"]; ok {
			sb.WriteString(fmt.Sprintf("Special Defense: %d\n", spDefense))
		}
		if speed, ok := pokemon.Stats["Speed"]; ok {
			sb.WriteString(fmt.Sprintf("Speed: %d\n", speed))
		}
		if total, ok := pokemon.Stats["Total"]; ok {
			sb.WriteString(fmt.Sprintf("Total: %d\n", total))
		}
		sb.WriteString("\n")
	}
	
	// Type Effectiveness
	if len(pokemon.WeakAgainst) > 0 || len(pokemon.StrongAgainst) > 0 {
		sb.WriteString("=== Type Effectiveness ===\n")
		if len(pokemon.WeakAgainst) > 0 {
			sb.WriteString(fmt.Sprintf("Weak against: %s\n", strings.Join(pokemon.WeakAgainst, ", ")))
		}
		if len(pokemon.StrongAgainst) > 0 {
			sb.WriteString(fmt.Sprintf("Strong against: %s\n", strings.Join(pokemon.StrongAgainst, ", ")))
		}
		sb.WriteString("\n")
	}
	
	// Evolution
	if len(pokemon.Evolutions) > 0 {
		sb.WriteString("=== Evolution Chain ===\n")
		sb.WriteString(fmt.Sprintf("Evolves to/from: %s\n", strings.Join(pokemon.Evolutions, " → ")))
		sb.WriteString("\n")
	}
	
	// Additional context for Q&A
	sb.WriteString("=== Quick Facts ===\n")
	sb.WriteString(fmt.Sprintf("- %s is a %s type Pokemon\n", pokemon.Name, strings.Join(pokemon.Types, "/")))
	if len(pokemon.Stats) > 0 {
		// Find highest stat
		maxStat := ""
		maxValue := 0
		for stat, value := range pokemon.Stats {
			if stat != "Total" && value > maxValue {
				maxValue = value
				maxStat = stat
			}
		}
		if maxStat != "" {
			sb.WriteString(fmt.Sprintf("- Highest stat: %s (%d)\n", maxStat, maxValue))
		}
	}
	if len(pokemon.Abilities) > 0 {
		sb.WriteString(fmt.Sprintf("- Primary ability: %s\n", pokemon.Abilities[0]))
	}
	
	return sb.String()
}
