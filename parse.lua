local query = require("ffluabase")

-- version:20191108-----2

-- 解析登录的用户名称
function ParseLoginUser(paymentid, html)
    local retObj = {}
    local doc, ok = query.newDocument(html)
    if ok then
        retObj["paymentid"] = paymentid
        local el, ok = query.find(doc, "#globalUser>span")
        if ok == false then
            return nil, false
        end
        local strName = query.text(el)
        local _, end_j, substr = string.find(strName, " ")
        strName = string.sub(strName,1, end_j)
        retObj["user_name"] = strName
    end
    return retObj, true
end

-- 解析收款记录
function ParseRecords(paymentid, html)
    local retObj = {}
    local recordArray = {}
    local doc, ok = query.newDocument(html)
    if ok then
        local el, ok = query.find(doc, "#tradeRecordsIndex>tbody>tr")
        if ok then
            local els, ok = query.each(el)
            if ok then
                for k, v in pairs(els) do
                    local record = {}
                    local elDate, ok = query.find(v, ".time>.time-d")
                    if ok == false then
                        return nil, false
                    end

                    local elTime, ok = query.find(v, ".time>.time-h")
                    if ok == false then
                        return nil, false
                    end

                    local elTrade, ok = query.find(v, ".tradeNo")
                    if ok == false then
                        return nil, false
                    end

                    local elAmount, ok = query.find(v, ".amount>.amount-pay")
                    if ok == false then
                        return nil, false
                    end

                    local elStatus, ok = query.find(v, ".status>p")
                    if ok == false then
                        return nil, false
                    end
                    -- print(m.text(elDate), m.text(elTime), m.text(elTrade), m.text(elAmount), m.text(elStatus))
                    record["paymentid"] = paymentid
                    record["date"] = query.text(elDate)
                    record["time"] = query.text(elTime)
                    record["trade_no"] = query.text(elTrade)
                    record["money"] = query.text(elAmount)
                    record["status"] = query.text(elStatus)
                    table.insert(recordArray, record)
                end
            else
                return nil, false
            end
        else
            return nil, false
        end
    else
        return nil, false
    end
    retObj["records"] = recordArray
    return retObj, true
end