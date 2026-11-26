# NUMERIC(…, 2) values travel as PostgreSQL text ("101.50"). Rails renders
# BigDecimal in JSON with BigDecimal#to_s("F"): no trailing zeros, at least one
# fraction digit ("101.5", "101.0"). Sums are kept in integer cents.
module Money
  def self.decimal_text(text)
    format_cents(cents(text))
  end

  def self.cents(text)
    negative = text.start_with?("-")
    digits = negative ? text[1, text.size - 1] : text
    dot = digits.index(".")
    whole = dot.nil? ? digits : digits[0, dot]
    frac = dot.nil? ? "" : digits[dot + 1, digits.size - dot - 1]
    frac = "#{frac}00"[0, 2]
    value = whole.to_i * 100 + frac.to_i
    negative ? -value : value
  end

  def self.format_cents(cents)
    sign = cents < 0 ? "-" : ""
    abs = cents < 0 ? -cents : cents
    frac = abs % 100
    fraction = (frac / 10).to_s
    fraction = "#{fraction}#{frac % 10}" unless frac % 10 == 0
    "#{sign}#{abs / 100}.#{fraction}"
  end
end
