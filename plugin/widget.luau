--!nonstrict
-- Bar widget: word in the bar, meaning in the tooltip.
-- Click = notification, right click = force refresh.
local core = require("./lib/core.luau")

local current = { word = "…", meaning = "", err = nil }

local function render()
  barWidget.setGlyph("language")
  if current.err then
    barWidget.setText("wotd")
    barWidget.setTooltip({ { "Error", current.err } })
  else
    barWidget.setText(current.word)
    if current.meaning ~= "" then
      local rows = { { current.word, current.meaning } }
      for _, ex in ipairs(current.examples or {}) do
        table.insert(rows, { ex.text, ex.translation or "" })
      end
      barWidget.setTooltip(rows)
    else
      barWidget.clearTooltip()
    end
  end
end

local function refresh(force)
  core.fetch(force, function(s)
    current = s
    render()
  end)
end

function update()
  noctalia.setUpdateInterval(3600 * 1000) -- helper caches per day; hourly check catches midnight
  render()
  refresh(false)
end

function onClick()
  if current.err then
    noctalia.notifyError("Word of the Day", current.err)
  else
    noctalia.notify(current.word, current.meaning)
  end
end

function onRightClick()
  refresh(true)
end
