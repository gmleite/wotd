--!nonstrict
-- Shared by the bar widget and the desktop widget: runs the Go helper and
-- hands back a plain table { word, meaning, examples, wordLang, meaningLang, err }.
-- examples = list of { text, translation? }.
local M = {}

function M.fetch(force, cb)
  local wordLang = noctalia.getConfig("word_lang")
  local meaningLang = noctalia.getConfig("meaning_lang")
  local argv = {
    noctalia.expandPath(noctalia.getConfig("binary")),
    "-word", wordLang,
    "-meaning", meaningLang,
  }
  if force then
    table.insert(argv, "-refresh")
  end

  local function fail(msg)
    cb({ word = "…", meaning = "", examples = {}, wordLang = wordLang, meaningLang = meaningLang, err = msg })
  end

  local accepted = noctalia.runAsync(argv, function(result)
    if result.exitCode ~= 0 then
      fail("wotd exited with " .. tostring(result.exitCode) .. ": " .. result.stderr)
      return
    end
    local data, jerr = noctalia.json.decode(result.stdout)
    if data == nil then
      fail("bad JSON from wotd: " .. tostring(jerr))
    elseif data.error and data.error ~= "" then
      fail(data.error)
    else
      cb({
        word = data.word,
        meaning = data.meaning,
        examples = data.examples or {},
        wordLang = wordLang,
        meaningLang = meaningLang,
        err = nil,
      })
    end
  end)

  if not accepted then
    fail("could not start wotd (is it installed at the configured path?)")
  end
end

return M
