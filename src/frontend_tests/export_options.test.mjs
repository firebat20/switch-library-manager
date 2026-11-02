import assert from "node:assert/strict";
import {readFile} from "node:fs/promises";
import {createRequire} from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const XLSX = require("../frontend/lib/js/xlsx/xlsx.full.min.js");

test("index.html includes script tag for xlsx library", async () => {
    const html = await readFile(new URL("../frontend/index.html", import.meta.url), "utf8");
    assert.match(html, /src="lib\/js\/xlsx\/xlsx\.full\.min\.js"/);
});

test("app.js table footer includes both CSV and XLSX export buttons", async () => {
    const source = await readFile(new URL("../frontend/app.js", import.meta.url), "utf8");
    assert.match(source, /class="btn btn-link export-btn"\s+data-format="csv"/);
    assert.match(source, /class="btn btn-link export-btn"\s+data-format="xlsx"/);
});

test("vendored XLSX library creates and parses valid Excel spreadsheets with unicode characters", () => {
    const wb = XLSX.utils.book_new();
    const rows = [
        ["Title", "TitleId", "Notes"],
        ["ゼルダの伝説 ティアーズ オブ ザ キングダム", "0100f2c0115b6000", "Japanese characters"],
        ["Pokémon Legends: Arceus", "01001f5010dfa000", "Accented characters"]
    ];
    const ws = XLSX.utils.aoa_to_sheet(rows);
    XLSX.utils.book_append_sheet(wb, ws, "games");

    const buffer = XLSX.write(wb, {type: "buffer", bookType: "xlsx"});
    assert.ok(buffer.length > 0);
    // Verify PK zip magic number
    assert.equal(buffer[0], 0x50);
    assert.equal(buffer[1], 0x4B);

    const parsedWb = XLSX.read(buffer);
    assert.deepEqual(parsedWb.SheetNames, ["games"]);
    const parsedData = XLSX.utils.sheet_to_json(parsedWb.Sheets.games);
    assert.equal(parsedData.length, 2);
    assert.equal(parsedData[0].Title, "ゼルダの伝説 ティアーズ オブ ザ キングダム");
    assert.equal(parsedData[1].Title, "Pokémon Legends: Arceus");
});

