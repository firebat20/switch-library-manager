const { shell, dialog } = require('electron').remote

$(function () {

    let state = {
        settings:{},
        keys:false
    };

    let currTable

    // Fluent UI formatter for Title + Thumbnail
    const fluentTitleFormatter = function(cell, formatterParams, onRendered){
        const data = cell.getRow().getData();
        const imgSrc = data.icon || (data.Attributes && data.Attributes.bannerUrl) || null;
        const title = data.name || (data.Attributes && data.Attributes.name) || 'Unknown Title';
        
        if (imgSrc) {
            return `<div style="display:flex; align-items:center; gap: 12px; padding: 4px 0;">
                      <img src="${imgSrc}" style="width: 52px; height: 52px; border-radius: 6px; object-fit: cover; box-shadow: 0 2px 6px rgba(0,0,0,0.15);">
                      <div style="font-weight: 600; font-size: 14px; white-space: normal; line-height: 1.3; color: var(--fluent-text, inherit);">${title}</div>
                    </div>`;
        } else {
            return `<div style="font-weight: 600; font-size: 14px; white-space: normal; padding: 4px 0; color: var(--fluent-text, inherit);">${title}</div>`;
        }
    };

    // Fluent UI formatter for File paths
    const fluentFileFormatter = function(cell, formatterParams, onRendered){
        const fullPath = cell.getValue();
        if (!fullPath) return "";

        // Try to split the path by standard slashes to separate filename from directory
        const normalizedPath = fullPath.replace(/\\/g, '/');
        const parts = normalizedPath.split('/');
        
        const fileName = parts.pop();
        const dirName = parts.join('/') || "/";
        
        return `<div style="display:flex; flex-direction:column; justify-content:center; padding: 4px 0; cursor: pointer;">
                  <div style="font-weight: 500; font-size: 13px; color: var(--fluent-text, inherit); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;" title="${fileName}">${fileName}</div>
                  <div style="font-weight: 400; font-size: 11px; color: #888; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;" title="${dirName}">${dirName}</div>
                </div>`;
    };

    //handle tabs action
    $('.tabgroup > div').hide();
    // loadTab($('.tabgroup > div:first-of-type'));

    // This will wait for the astilectron namespace to be ready
    document.addEventListener('astilectron-ready', function () {
        
        // Restore Maximized State from backend settings
        astilectron.sendMessage({name: "checkMaximized", payload: ""}, function(message) {
            if (message === "true") {
                try { require('electron').remote.getCurrentWindow().maximize(); } catch(e){}
            }
        });

        let sendMessage = function (name, payload, callback) {
            astilectron.sendMessage({name: name, payload: payload}, callback)
        };

        sendMessage("loadSettings", "", function (message) {
            state.settings = JSON.parse(message);

            if(state.settings.hide_missing_games){
                document.getElementById("tab_btns").classList.add("hide_missing_games");
            }

            // Apply Dark Mode from settings
            if(state.settings.dark_mode) {
                document.body.classList.add("bootstrap-dark");
                document.body.classList.remove("bootstrap");
                try { require('electron').remote.nativeTheme.themeSource = 'dark'; } catch(e){}
                $('meta[name="color-scheme"]').attr("content", "dark");
                $("#toggle-dark-mode").text("☀️");
            } else {
                document.body.classList.add("bootstrap");
                document.body.classList.remove("bootstrap-dark");
                try { require('electron').remote.nativeTheme.themeSource = 'light'; } catch(e){}
                $('meta[name="color-scheme"]').attr("content", "light");
                $("#toggle-dark-mode").text("🌙");
            }
        });

        sendMessage("isKeysFileAvailable", "", function (message) {
            state.keys = message
        });

        sendMessage("checkUpdate", "", function (message) {
            if (message === "false"){
                return
            }
            dialog.showMessageBox(null, {
                type: 'info',
                buttons: ['Ok'],
                defaultId: 0,
                title: 'New update available',
                message: 'There is a new update available, please download from Github',
                detail: message.payload
            });
        });

        $(".progress-container").show();
        $(".progress-type").text("Downloading latest Switch titles/versions ...");

        sendMessage("updateDB", "", function (message) {
            scanLocalFolder();
        });

        astilectron.onMessage(function (message) {
            // Process message
            // console.log(message)
            let pcg = 0
            if (message.name === "updateProgress") {
                let pp = JSON.parse(message.payload);
                let count = pp.curr;
                let total = pp.total;
                $('.progress-msg').text(pp.message);
                if (count !== -1 && total !== -1){
                    pcg = Math.floor(count / total * 100);
                    $('.progress-bar').attr('aria-valuenow', pcg);
                    $('.progress-bar').attr('style', 'width:' + Number(pcg) + '%');
                    $('.progress-bar').text(pcg + "%");
                }
                if (pcg === 100){
                    $(".progress-container").hide();
                }else{
                    $(".progress-container").show();
                }
            }
            else if (message.name === "libraryLoaded") {
                state.library = JSON.parse(message.payload);
                loadTab("#library")
            }
            else if (message.name === "missingGames") {
                state.missingGames = JSON.parse(message.payload);
                loadTab("#missing")
            }
            else if (message.name === "error") {
                dialog.showMessageBox(null, {
                    type: 'error',
                    buttons: ['Ok'],
                    defaultId: 0,
                    title: 'Error',
                    message: 'An unexpected error occurred',
                    detail: message.payload
                });
                state.settings.folder = undefined;
                $(".progress-container").hide();
                loadTab("#library")
            }
            else if (message.name === "rescan") {
                state.library = undefined;
                state.updates = undefined;
                state.dlc = undefined;
                scanLocalFolder(true)
            }
        });

        let openFolderPicker = function (mode) {
            //show info
            dialog.showOpenDialog({
                properties: ['openDirectory'],
                message:"Select games folder"
            }).then(partial(updateFolder,mode))
                .catch(error => console.log(error))
        };

        let scanLocalFolder = function(mode){
            if (!state.settings.folder){
                loadTab("#library")
                return
            }
            //show progress
            $(".progress-container").show();
            $(".progress-type").text("Scanning local library...");

            sendMessage("updateLocalLibrary", ""+mode, (r => {}))
        };

        let updateFolder = function (mode,result) {
            if (result.canceled) {
                console.log("user aborted");
                return
            }
            if (!result.filePaths || !result.filePaths.length){
                return
            }

            if (mode === "add"){
                state.settings.scan_folders = state.settings.scan_folders || []
                if (!state.settings.scan_folders.includes(result.filePaths[0])){
                    state.settings.scan_folders.push(result.filePaths[0]);
                }else{
                    return;
                }

            }else{
                state.settings.folder = result.filePaths[0];
            }
            $('.tabgroup > div').hide();
            console.log("selected folder:"+result.filePaths[0]);
            state.library = undefined;
            state.updates = undefined;
            state.dlc = undefined;
            sendMessage("saveSettings", JSON.stringify(state.settings), scanLocalFolder);
        };


        function loadTab(target) {
            hideCurrentTab();

            $("#tab_btns a[href='" + target + "']").addClass('active');
            $(target).show();

            if (target === "#settings") {
                let settingsJSON = JSON.stringify(state.settings, null, 2)
                let settingsHtml = $(target + "Template").render({code: settingsJSON})
                $(target).html(settingsHtml);
                //  asticode.loader.hide()
            } else if (target === "#organize") {
                let html = $(target + "Template").render({folder: state.settings.folder,settings:state.settings})
                $(target).html(html);
            } else if (target === "#updates") {
                if (state.settings.folder && !state.library){
                    return
                }
                if (state.library && !state.updates){
                    sendMessage("missingUpdates", "", (r => {
                        state.updates = JSON.parse(r)
                        loadTab("#updates")
                    }));
                    return
                }
                let html = $(target + "Template").render({folder: state.settings.folder,updates:state.updates})
                $(target).html(html);
                if (state.updates && state.updates.length) {
                    currTable = new Tabulator("#updates-table", {
                        layout:"fitDataStretch",
                        initialSort:[
                            {column:"latest_update_date", dir:"desc"}, //sort by this first
                        ],
                        pagination: "local",
                        paginationSize: state.settings.gui_page_size,
                        data: state.updates,
                        columns: [
                            {title: "Game", field: "Attributes.name", headerFilter:"input", formatter:fluentTitleFormatter, width:400},
                            {title: "Type", field: "Meta.type", headerFilter:"input"},
                            {title: "Title ID", headerSort:false, field: "Attributes.id", hozAlign: "right", sorter: "number"},
                            {title: "Local version", headerSort:false, field: "local_update", hozAlign: "right", sorter: "number"},
                            {title: "Available version", headerSort:false, field: "latest_update", hozAlign: "right"},
                            {title: "Update date", headerSort:true, field: "latest_update_date",sorter:"date", sorterParams:{format:"YYYY-MM-DD"}}
                        ],
                    });
                }
            } else if (target === "#dlc") {
                if (state.settings.folder && !state.library){
                    return
                }
                if (state.library && !state.dlc){
                    sendMessage("missingDlc", "", (r => {
                        state.dlc = JSON.parse(r)
                        loadTab("#dlc")
                    }));
                    return
                }
                let html = $(target + "Template").render({folder: state.settings.folder,dlc:state.dlc});
                $(target).html(html);
                if (state.dlc && state.dlc.length) {
                    currTable = new Tabulator("#dlc-table", {
                        layout:"fitDataStretch",
                        initialSort:[
                            {column:"Attributes.name", dir:"asc"}, //sort by this first
                        ],
                        pagination: "local",
                        paginationSize: state.settings.gui_page_size,
                        data: state.dlc,
                        columns: [
                            {title: "Game", field: "Attributes.name", headerFilter:"input",formatter:fluentTitleFormatter, width:400},
                            {title: "# Missing", field: "missing_dlc.length"},
                            {title: "Missing DLC", headerSort:false, field: "missing_dlc",formatter:function(cell, formatterParams, onRendered){
                                    value = ""
                                    for (var i in cell.getValue())
                                    {
                                        value +="<div>"+cell.getValue()[i]+"</div>"
                                    }
                                    return value
                                }}
                        ],
                    });
                }
            } else if (target === "#status") {
                if (state.settings.folder && !state.library){
                    return
                }
                let html = $(target + "Template").render({folder: state.settings.folder,library:state.library ? state.library.issues: undefined,numFiles:state.library ? state.library.num_files:-1});
                $(target).html(html);
                if (state.library.issues && state.library.issues.length) {
                    currTable = new Tabulator("#status-table", {
                        layout:"fitDataStretch",
                        pagination: "local",
                        paginationSize: state.settings.gui_page_size,
                        data: state.library.issues,
                        columns: [
                            {title: "File name",width:500, headerSort:false, field: "key",formatter:fluentFileFormatter,cellClick:function(e, cell){
                                    //e - the click event object
                                    //cell - cell component
                                    shell.showItemInFolder(cell.getData().key)
                                }
                            },
                            {
                                title: "Issue", field: "value", formatter: function (cell) {
                                    return cell.getValue()
                                        .replaceAll("\nNew: ", "<br/><strong style='color:#0078D4; margin-top:8px; display:inline-block'>New:</strong> ")
                                        .replaceAll("\nOld: ", "<br/><strong style='color:#E81123; margin-top:4px; display:inline-block'>Old:</strong> ")
                                        .replaceAll("\nExisting: ", "<br/><strong style='color:#0078D4; margin-top:8px; display:inline-block'>Existing:</strong> ")
                                        .replaceAll("\nDuplicate: ", "<br/><strong style='color:#E81123; margin-top:4px; display:inline-block'>Duplicate:</strong> ")
                                        .replaceAll("\n", "<br/>");
                                }
                            }
                        ],
                    });
                }
            } else if (target === "#library") {
                if (state.settings.folder && !state.library){
                    return
                }
                let html = $(target + "Template").render(
                    {
                        folder: state.settings.folder,
                        library: state.library ? state.library.library_data : [] ,
                        num_skipped:state.library ? (state.library.issues ? state.library.issues.length : 0) : 0,
                        num_files:state.library ? state.library.num_files : 0,
                        keys:state.keys,
                        scanFolders:state.settings.scan_folders
                    })
                $(target).html(html);
                if (state.library && state.library.library_data.length) {
                    currTable = new Tabulator("#library-table", {
                        initialSort:[
                            {column:"name", dir:"asc"}, //sort by this first
                        ],
                        layout:"fitDataStretch",
                        pagination: "local",
                        paginationSize: state.settings.gui_page_size,
                        data: state.library.library_data,
                        columns: [
                            {title: "Game", field: "name", headerFilter:"input", formatter:fluentTitleFormatter, width:400},
                            {title: "Title ID", headerSort:false, field: "titleId"},
                            {title: "Region", headerSort:true, field: "region"},
                            {title: "Type", headerSort:true, field: "type"},
                            {title: "Update", headerSort:false, field: "update"},
                            {title: "Version", headerSort:false, field: "version"},
                            {title: "File name", headerSort:false, field: "path",formatter:fluentFileFormatter,cellClick:function(e, cell){
                                    //e - the click event object
                                    //cell - cell component
                                    shell.showItemInFolder(cell.getData().path)
                                }
                            }
                        ],
                    });
                }
            } else if (target === "#missing") {
                if (state.settings.folder && !state.library){
                    return
                }
                if (state.library && !state.missingGames){
                    sendMessage("missingGames", "", (r => {
                        state.missingGames = JSON.parse(r)
                        loadTab("#missing")
                    }));
                    return
                }
                let html = $(target + "Template").render({folder: state.settings.folder,missingGames:state.missingGames});
                $(target).html(html);
                if (state.missingGames && state.missingGames.length) {
                    currTable = new Tabulator("#missingGames-table", {
                        layout:"fitDataStretch",
                        initialSort:[
                            {column:"name", dir:"asc"}, //sort by this first
                        ],
                        pagination: "local",
                        paginationSize: state.settings.gui_page_size,
                        data: state.missingGames,
                        columns: [
                            {field: "name", title: "Game", headerFilter:"input", formatter:fluentTitleFormatter, width:400},
                            {title: "Title ID", headerSort:false, field: "titleId"},
                            {title: "Region", headerSort:true,headerFilter:"input",formatter:"textarea", field: "region"},
                            {title: "Release date", headerSort:true, field: "release_date", sorter:"date", sorterParams:{format:"YYYY-MM-DD"}},
                        ],
                    });
                }
            }
        }

        $("body").on("click", ".folder-set", e => {
            openFolderPicker(e.target.textContent.toLowerCase().trim())
        });

        $("body").on("click", ".export-btn", e => {
            currTable.download("csv", "export.csv", {}, "all");
        });

        // Dark Mode Toggle
        $("body").on("click", "#toggle-dark-mode", e => {
            e.preventDefault();
            state.settings.dark_mode = !state.settings.dark_mode;
            
            if(state.settings.dark_mode) {
                document.body.classList.add("bootstrap-dark");
                document.body.classList.remove("bootstrap");
                try { require('electron').remote.nativeTheme.themeSource = 'dark'; } catch(e){}
                $('meta[name="color-scheme"]').attr("content", "dark");
                $("#toggle-dark-mode").text("☀️");
            } else {
                document.body.classList.add("bootstrap");
                document.body.classList.remove("bootstrap-dark");
                try { require('electron').remote.nativeTheme.themeSource = 'light'; } catch(e){}
                $('meta[name="color-scheme"]').attr("content", "light");
                $("#toggle-dark-mode").text("🌙");
            }
            
            // Save the toggle preference without scanning
            sendMessage("saveSettings", JSON.stringify(state.settings), function(){});
        });

        // Rescan Library Toggle
        $("body").on("click", "#btn-rescan", e => {
            e.preventDefault();
            state.library = undefined;
            state.updates = undefined;
            state.dlc = undefined;
            scanLocalFolder(true);
        });

        // Hard Rescan Toggle
        $("body").on("click", "#btn-hard-rescan", e => {
            e.preventDefault();
            const options = {
                type: 'warning',
                buttons: ['Yes', 'No'],
                defaultId: 0,
                title: 'Confirmation',
                message: 'Are you sure you want to perform a Hard Rescan?',
                detail: 'This will completely clear the local database cache and do a deep scan of all your files again. It will take longer than a normal rescan.',
            };
            dialog.showMessageBox(null, options).then( (r) => {
                if (r.response === 0) {
                    sendMessage("hardRescan", "", function(){});
                }
            });
        });

        $("body").on("click", ".library-organize-action", e => {
            e.preventDefault();
            if (state.settings.organize_options.create_folder_per_game === false &&
                state.settings.organize_options.rename_files === false){
                dialog.showMessageBox(null, {
                    type: 'info',
                    buttons: ['Ok'],
                    defaultId: 0,
                    title: 'Library organization is turned off',
                    message: 'Please update settings.json to enable this feature',
                    detail: "You should set 'rename_files' and/or 'create_folder_per_game' to 'true' "
                });
                return
            }
            const options = {
                type: 'warning',
                buttons: ['Yes', 'No'],
                defaultId: 0,
                title: 'Confirmation',
                message: 'Are you sure you want to begin library organization?',
                detail: 'This action will modify your local library files',
            };

            dialog.showMessageBox(null, options).then( (r) => {

                if (r.response === 0) {
                    //show progress
                    $('.tabgroup > div').hide();
                    $(".progress-container").show();
                    $(".progress-type").text("Organizing local library...");

                    sendMessage("organize", "", (r => {
                        $(".progress-container").hide();
                        state.library = undefined;
                        state.updates = undefined;
                        state.dlc = undefined;
                        loadTab("#library");
                        scanLocalFolder(true)
                        dialog.showMessageBox(null, {
                            type: 'info',
                            buttons: ['Ok'],
                            defaultId: 0,
                            title: 'Success',
                            message: 'Operation completed successfully'
                        })
                    }))
                }
            });

        });

        $('#tab_btns a').click(function (e) {
            e.preventDefault();
            let target = $(e.currentTarget).attr('href');
            if (target === "#") return; // Ignore icon buttons in navbar
            loadTab(target);
        });

        function hideCurrentTab() {
            $("#tab_btns a").removeClass("active");
            let tabgroup = $("#tab_btns").data('tabgroup');
            $("#" + tabgroup).children('div').hide();
        }

        function partial(func /*, 0..n args */) {
            var args = Array.prototype.slice.call(arguments, 1);
            return function() {
                var allArguments = args.concat(Array.prototype.slice.call(arguments));
                return func.apply(this, allArguments);
            };
        }

        // Track Window Dimensions
        let resizeTimer;
        window.addEventListener('resize', () => {
            clearTimeout(resizeTimer);
            resizeTimer = setTimeout(() => {
                try {
                    const win = require('electron').remote.getCurrentWindow();
                    const bounds = win.getBounds();
                    const isMax = win.isMaximized();
                    state.settings.window_maximized = isMax;
                    if (!isMax) {
                        state.settings.window_width = bounds.width;
                        state.settings.window_height = bounds.height;
                    }
                    sendMessage("saveSettings", JSON.stringify(state.settings), function(){});
                } catch(e) {}
            }, 1000); // Save bounds 1 second after user finishes resizing
        });

    });

});