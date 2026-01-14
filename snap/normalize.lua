local M = {}

local function normalize_list(value, key_field)
  if type(value) ~= "table" then
    return {}
  end
  local list = {}
  for key, item in pairs(value) do
    if type(item) == "table" then
      local entry = item
      if entry[key_field] == nil and type(key) == "number" then
        entry = vim.deepcopy(entry)
        entry[key_field] = key
      end
      table.insert(list, entry)
    end
  end
  table.sort(list, function(a, b)
    return (a[key_field] or 0) < (b[key_field] or 0)
  end)
  return list
end

local function normalize_groups(value)
  if type(value) ~= "table" then
    return {}
  end
  local list = {}
  for _, item in pairs(value) do
    if type(item) == "table" then
      table.insert(list, item)
    end
  end
  table.sort(list, function(a, b)
    return tostring(a.name or "") < tostring(b.name or "")
  end)
  return list
end

function M.normalize(snapshot)
  if type(snapshot) ~= "table" then
    return {}
  end
  local normalized = {}
  for key, value in pairs(snapshot) do
    normalized[key] = value
  end

  normalized.grids = normalize_list(snapshot.grids, "id")
  normalized.hl_attrs = normalize_list(snapshot.hl_attrs, "id")
  normalized.hl_groups = normalize_groups(snapshot.hl_groups)

  return normalized
end

return M
