package filters

import log "github.com/cihub/seelog"
import "regexp"
import "time"
import "github.com/cwacek/irengine/scanner/filereader"
import "strconv"
import "fmt"
import "strings"

type DateFilterState int

const (
  DateBegin = 0
  DateDayWeek = iota
  DateMonth
  DateDayMonth
  DateTime
  DateYear
)

func (d DateFilterState) String() string {
  switch d {
  case DateBegin:
    return "DateBegin"
  case DateDayWeek:
    return "DateDayWeek"
  case DateMonth:
    return "DateMonth"
  case DateDayMonth:
    return "DateDayMonth"
  case DateTime:
    return "DateTime"
  case DateYear:
    return "DateYear"
  default:
    return "Unknown"
  }
}

var (
  TimeFormats = []string{
    "15:04:05",
    "15:04",
    "3:04pm",
    "3:04:05pm",
    "3:04PM",
    "3:04:05PM",
    "3PM",
    "3pm",
  }

  YearRegex = regexp.MustCompile(`[\d]{2}[\d]{2}?$`)
  DateRegex = regexp.MustCompile(`(1[0-2]|[1-9]|0[1-9])[-\/]([0-9]|[0-3][0-9]?)[-\/]([\d]{2}|[\d]{4})$`)

  Months = map[string]time.Month {
    "january": 1,
    "jan": 1,
    "february": 2,
    "feb": 2,
    "march": 3,
    "mar": 3,
    "april": 4,
    "apr": 4,
    "may": 5,
    "june": 6,
    "jun": 6,
    "july": 7,
    "jul": 7,
    "august": 8,
    "aug": 8,
    "september": 9,
    "sep": 9,
    "october": 10,
    "oct": 10,
    "november": 11,
    "nov":11,
    "december": 12,
    "dec": 12,
  }


)

type DateFilter struct {
  FilterPlumbing

  havePartial bool
  matches map[DateFilterState]string
  state DateFilterState
}

func NewDateFilter(id string) Filter {
  f := new(DateFilter)
  f.Id = id
  f.self = f
  f.havePartial = false
  f.matches = make(map[DateFilterState]string)
  f.state = DateBegin
  return f
}

func (f *DateFilter) Reset() {
  f.matches = make(map[DateFilterState]string)
  f.state = DateBegin
  f.havePartial = false
}

func (f *DateFilter) Apply(tok *filereader.Token) (result []*filereader.Token) {
  var (
    newtok *filereader.Token
    ok bool
  )
  result = make([]*filereader.Token, 0, 1)

ParseAgain:

  log.Debugf("Datefilter read %s in state %v", tok, f.state)
  switch f.state {
  case DateBegin:
    newtok, ok = f.TryMatchDate(tok)

    if ok {
      result = append(result, newtok)
      f.Reset()
      break
    }

    newtok, ok = f.TryMatchMonth(tok)
    if ok {
      // Include the month
      result = append(result, newtok)
      f.state = DateMonth
      break
    }

    result = append(result, tok)

  case DateMonth:
    newtok, ok = f.TryMatchDay(tok)
    if ok {
      //Don't include the day
      log.Debugf("State DateMonth: %s matched day", tok)
      f.state = DateDayMonth
      break
    }

    newtok, ok = f.TryMatchYear(tok)
    if ok {
      // Include the year separately
      result = append(result, newtok)
      //This ends a date representation
      dateTok := CloneWithText(tok, f.makeDateRepr())
      dateTok.Final = true
      result = append(result, dateTok)
      f.Reset()
      break
    }

    log.Debugf("State DateMonth: %s matched nothing", tok)
    // If we didn't match either, this is probably just a Month, 
    // so let's just push the token along and reset.
    result = append(result, tok)
    f.Reset()

  case DateDayMonth:
    newtok, ok = f.TryMatchYear(tok)
    if ok {
      // Include the year separately
      result = append(result, tok)
      //This ends a date representation
      dateTok := CloneWithText(tok, f.makeDateRepr())
      result = append(result, dateTok)
      f.Reset()
      break
    }

    // If we've seen a month, and this isn't a year, it's probably
    // the end of the date, so scrap it.
    dateTok := CloneWithText(tok, f.makeDateRepr())
    dateTok.Final = true
    result = append(result, dateTok)
    f.Reset()
    goto ParseAgain
    break


  default:
    result = append(result, tok)
  }
  return
}

func (f *DateFilter) makeDateRepr() string {
  var day, month, year string
  var ok bool

  if day, ok = f.matches[DateDayMonth]; ! ok {
    day = "00"
  }

  if month, ok = f.matches[DateMonth]; ! ok {
    month = "00"
  }

  if year, ok = f.matches[DateYear]; ! ok {
    year = "0000"
  }

  return fmt.Sprintf("%s_%s_%s", month, day, year)
}

func (f *DateFilter) TryMatchMonth(tok *filereader.Token) (*filereader.Token, bool) {
  if match, ok := Months[strings.ToLower(tok.Text)]; ok {
    f.matches[DateMonth] = fmt.Sprintf("%02d", match)
    f.havePartial = true
    // We return tok so the month name gets added separately.
    newtok := CloneWithText(tok, match.String())
    return newtok, true
  }
  return nil, false
}

func (f *DateFilter) TryMatchYear(tok *filereader.Token) (*filereader.Token, bool) {
  if num, err := strconv.Atoi(tok.Text); err == nil {
    f.matches[DateYear] = fmt.Sprintf("%04d", num)
    f.havePartial = true
    return tok, true
  }
  return nil, false
}

func (f *DateFilter) TryMatchDay(tok *filereader.Token) (*filereader.Token, bool) {
  log.Debugf("Trying to match day on %s", tok)
  daystr := strings.TrimRight(tok.Text, "thstrd")
  log.Debugf("After trimming, have %s", daystr)
  if num, err  := strconv.Atoi(daystr); err == nil {

    if num > 0 && num < 31 {
      f.matches[DateDayMonth] = fmt.Sprintf("%02d", num)
      f.havePartial = true
      return nil, true
    }
  }
  return nil, false
}

func (f *DateFilter) TryMatchDate(tok *filereader.Token) (*filereader.Token, bool) {

  log.Debugf("Trying to match date on %s", tok)

  if m := DateRegex.FindStringSubmatch(tok.Text); m != nil {
    // m is a slice with [full, month, day, year]
    log.Debugf("Matched with month %s, day %s, year %s", m[1], m[2], m[3])
    if monthIdx, err := strconv.Atoi(m[1]); err == nil {
      if monthIdx > 0 && monthIdx <= 12 {
        f.matches[DateMonth] = fmt.Sprintf("%02d", monthIdx)
      } else {
        log.Debugf("Month index %d wasn't in the right range", monthIdx)
        return nil, false
      }
    } else {
      log.Debugf("Failed to convert %s to month: %s", m[1], err)
    }

    if num, err := strconv.Atoi(m[2]); err == nil {
      if num > 0 && num <= 31 {
        f.matches[DateDayMonth] = fmt.Sprintf("%02d", num)
      } else {
        log.Debugf("Day index %d wasn't in the right range", num)
        return nil, false
      }
    } else {
      log.Debugf("Failed to convert %s to day: %s", m[2], err)
    }

    if year, err := strconv.Atoi(m[3]); err == nil {
      nowYear := time.Now().Year()

      log.Debugf("Comparing %d to nowyear %d", year, nowYear)
      switch {
      case year < 100 && year <= (nowYear / 100):
        // This is a 20XX number
        f.matches[DateYear] = fmt.Sprintf("%02d%02d", nowYear/100, year)

      case year < 100 && year > (nowYear / 100):
        // This is a 19XX number
        f.matches[DateYear] = fmt.Sprintf("%02d%02d", ( nowYear/100 ) - 1, year)

      case year > 100:
        f.matches[DateYear] = fmt.Sprintf("%04d", year)

      default:
        log.Debugf("Couldn't intepret year %d", year)

      }
    } else {
      log.Debugf("Failed to convert %s to year: %s", m[3], err)
    }

    newtok := CloneWithText(tok,
    fmt.Sprintf("%s_%s_%s", f.matches[DateMonth],
    f.matches[DateDayMonth], f.matches[DateYear]))
    newtok.Final = true
    return newtok, true
  }
  log.Debugf("Failed to match %s as date", tok)
  return nil, false
}
