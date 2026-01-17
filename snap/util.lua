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

function M.text_diff(expected, actual, opts)
  if vim.text and vim.text.diff then
    return vim.text.diff(expected, actual, opts)
  end
  if not vim.diff then
    return nil
  end
  local ok, result = pcall(vim.diff, expected, actual, opts)
  if ok then
    return result
  end
  local fallback = {
    result_type = opts and opts.result_type or nil,
    ctxlen = opts and opts.ctxlen or nil,
  }
  local ok2, result2 = pcall(vim.diff, expected, actual, fallback)
  if ok2 then
    return result2
  end
  return nil
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
