local M = {}

local unpack_fn = table.unpack or unpack

function M.rpc_error_to_string(err)
  if err == nil or err == vim.NIL then
    return "unknown error"
  end
  if type(err) == "table" then
    return vim.inspect(err)
  end
  return tostring(err)
end

function M.err_write(msg)
  if msg == nil then
    msg = "unknown error"
  end
  io.stderr:write(tostring(msg), "\n")
end

function M.joinpath(...)
  local parts = { ... }
  local filtered = {}
  for _, part in ipairs(parts) do
    if part ~= nil and part ~= "" then
      table.insert(filtered, tostring(part))
    end
  end
  if #filtered == 0 then
    return ""
  end
  if vim.fs and vim.fs.joinpath then
    return vim.fs.joinpath(unpack_fn(filtered))
  end
  return table.concat(filtered, "/")
end

function M.normalize_path(base, path)
  if not path then
    return nil
  end
  if vim.fn.fnamemodify(path, ":p") == path then
    return vim.fs.normalize(path)
  end
  return vim.fs.normalize(M.joinpath(base, path))
end

return M
