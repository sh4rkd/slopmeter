import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Window {
    id: craftMarketWindow
    objectName: "craftMarketWindow"
    visible: false
    width: 620; height: 720; minimumWidth: 480; minimumHeight: 400
    color: "#151e2b"; flags: Qt.Window
    title: "SlopMeter · Calculadora de Economía y Crafteo Aion 2"

    property string categoryFilter: "TODOS"
    property string searchText: ""
    property string excludeText: ""
    property bool onlyProfitable: false
    property string sortBy: "profit" // "profit", "roi", "sellPrice", "cost", "name", "budgetProfit"
    property bool sortAscending: false
    property double availableBudget: 0.0

    function formatKinah(value) {
        if (value === undefined || value === null || isNaN(value)) return "0"
        const num = Number(value)
        if (num === 0) return "0"
        const sign = num < 0 ? "-" : ""
        const absVal = Math.abs(Math.round(num))
        if (absVal >= 1000000000) {
            return sign + (absVal / 1000000000).toFixed(2) + "B"
        } else if (absVal >= 1000000) {
            return sign + (absVal / 1000000).toFixed(2) + "M"
        } else if (absVal >= 10000) {
            return sign + (absVal / 1000).toFixed(1) + "K"
        } else {
            return sign + absVal.toLocaleString(Qt.locale(), 'f', 0)
        }
    }

    ColumnLayout {
        anchors.fill: parent; anchors.margins: 14; spacing: 8

        // Header
        RowLayout {
            Layout.fillWidth: true; spacing: 8
            Label {
                text: "⚒ ECONOMÍA Y CRAFTEO AION 2"
                color: "#f1c778"; font.pixelSize: 15; font.bold: true
                Layout.fillWidth: true
            }
            ComboBox {
                id: serverCombo
                model: backend.marketServers
                textRole: "name"
                valueRole: "id"
                implicitWidth: 160; implicitHeight: 28
                currentIndex: {
                    for (let i = 0; i < backend.marketServers.length; ++i) {
                        if (backend.marketServers[i].id === backend.selectedMarketServer) return i
                    }
                    return 0
                }
                onActivated: backend.selectedMarketServer = currentValue
            }
            Button {
                text: backend.marketLoading ? "Cargando..." : "↻ Actualizar Precios"
                enabled: !backend.marketLoading
                implicitHeight: 28
                palette.button: "#24374d"
                palette.buttonText: "#54cf82"
                onClicked: backend.fetchMarketPrices()
            }
        }

        // Status bar
        Rectangle {
            Layout.fillWidth: true; implicitHeight: 28; radius: 3
            color: "#1c2738"; border.width: 1; border.color: "#2a3d57"
            RowLayout {
                anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8; spacing: 8
                Label {
                    text: backend.marketStatus
                    color: "#9ad9ec"; font.pixelSize: 10
                    Layout.fillWidth: true; elide: Text.ElideRight
                }
                Label {
                    text: "Tasa Impuesto Broker:"
                    color: "#f1c778"; font.pixelSize: 10; font.bold: true
                }
                SpinBox {
                    id: taxSpinBox
                    from: 0; to: 30; stepSize: 1
                    implicitWidth: 85; implicitHeight: 24
                    value: Math.round(backend.marketTaxRate)
                    textFromValue: function(value) { return value + " %" }
                    valueFromText: function(text) { return parseInt(text) || 0 }
                    onValueModified: backend.marketTaxRate = value
                    ToolTip.visible: hovered; ToolTip.text: "Tasa de deducción del Broker en Aion 2 (Estándar: 10% / Varía con membresía)"
                }
            }
        }

        // Filtros y Búsqueda
        RowLayout {
            Layout.fillWidth: true; spacing: 8
            TextField {
                id: searchInput
                placeholderText: "Buscar receta o producto..."
                implicitHeight: 28
                Layout.fillWidth: true
                palette.text: "#ffffff"
                palette.base: "#1a2433"
                onTextChanged: craftMarketWindow.searchText = text.toLowerCase().trim()
            }
            TextField {
                id: excludeInput
                placeholderText: "Excluir material..."
                implicitHeight: 28
                Layout.preferredWidth: 140
                palette.text: "#ff9e9e"
                palette.base: "#241a1e"
                ToolTip.visible: hovered; ToolTip.text: "Excluye recetas que requieran este material"
                onTextChanged: craftMarketWindow.excludeText = text.toLowerCase().trim()
            }
            // Campo de Kinah Disponible / Presupuesto para Inversión
            Rectangle {
                implicitWidth: 170; implicitHeight: 28; radius: 3
                color: "#1c2a38"; border.width: 1; border.color: craftMarketWindow.availableBudget > 0 ? "#54cf82" : "#354a63"
                RowLayout {
                    anchors.fill: parent; anchors.margins: 4; spacing: 4
                    Label {
                        text: "💰 Kinah Disp:"
                        color: "#f1c778"; font.pixelSize: 9; font.bold: true
                    }
                    TextField {
                        id: budgetField
                        Layout.fillWidth: true; Layout.fillHeight: true
                        placeholderText: "Ej. 5M, 500k, 1000000"
                        palette.text: "#54cf82"; palette.base: "transparent"
                        font.pixelSize: 10; font.bold: true
                        background: null
                        onTextChanged: {
                            let raw = text.trim().toLowerCase().replace(/,/g, '')
                            if (raw === "") {
                                craftMarketWindow.availableBudget = 0.0
                                return
                            }
                            let mult = 1.0
                            if (raw.endsWith('k')) { mult = 1000.0; raw = raw.slice(0, -1); }
                            else if (raw.endsWith('m')) { mult = 1000000.0; raw = raw.slice(0, -1); }
                            else if (raw.endsWith('b')) { mult = 1000000000.0; raw = raw.slice(0, -1); }
                            let parsed = parseFloat(raw)
                            craftMarketWindow.availableBudget = (!isNaN(parsed) && parsed > 0) ? (parsed * mult) : 0.0
                        }
                    }
                }
                ToolTip.visible: budgetHover.hovered; ToolTip.text: "Ingresa tu Kinah disponible para comprar ingredientes (ej. '10M' o '500k'). Slopmeter calculará cuánto puedes craftear y el beneficio máximo con tu dinero."
                HoverHandler { id: budgetHover }
            }
            CheckBox {
                text: "Solo Rentables"
                checked: craftMarketWindow.onlyProfitable
                palette.windowText: "#54cf82"
                onToggled: craftMarketWindow.onlyProfitable = checked
            }
        }

        // Barra de Ordenamiento (Ascendente / Descendente y Criterio)
        RowLayout {
            Layout.fillWidth: true; spacing: 8
            Label {
                text: "Ordenar por:"
                color: "#9ad9ec"; font.pixelSize: 10; font.bold: true
            }
            ComboBox {
                id: sortFieldCombo
                implicitWidth: 190; implicitHeight: 26
                model: [
                    { label: "Beneficio Neto (Kinah)", value: "profit" },
                    { label: "Beneficio con tu Presupuesto", value: "budgetProfit" },
                    { label: "Retorno (% ROI)", value: "roi" },
                    { label: "Precio de Venta", value: "sellPrice" },
                    { label: "Costo de Materiales", value: "cost" },
                    { label: "Nombre de Receta", value: "name" }
                ]
                textRole: "label"
                valueRole: "value"
                currentIndex: 0
                onActivated: craftMarketWindow.sortBy = currentValue
            }
            Rectangle {
                implicitWidth: sortOrderBtn.implicitWidth + 12; implicitHeight: 26; radius: 3
                color: "#213247"; border.width: 1; border.color: "#415f82"
                RowLayout {
                    id: sortOrderBtn
                    anchors.centerIn: parent; spacing: 4
                    Label {
                        text: craftMarketWindow.sortAscending ? "▲ Ascendente (Menor a Mayor)" : "▼ Descendente (Mayor a Menor)"
                        color: "#54cf82"; font.pixelSize: 10; font.bold: true
                    }
                }
                TapHandler {
                    onTapped: craftMarketWindow.sortAscending = !craftMarketWindow.sortAscending
                }
                HoverHandler { id: sortHover; cursorShape: Qt.PointingHandCursor }
                ToolTip.visible: sortHover.hovered; ToolTip.text: "Alternar orden ascendente o descendente"
            }
            Item { Layout.fillWidth: true }
            Label {
                text: recipeListView.count + " recetas encontradas"
                color: "#7e8ea6"; font.pixelSize: 9
            }
        }

        // Selector de Categoría de Crafteo
        ScrollView {
            Layout.fillWidth: true; implicitHeight: 32
            ScrollBar.vertical.policy: ScrollBar.AlwaysOff
            RowLayout {
                spacing: 4
                Repeater {
                    model: ["TODOS", "Alquimia", "Cocina", "Herrajes", "Artesanía", "Fabricación de armaduras", "Conversión"]
                    delegate: Rectangle {
                        required property string modelData
                        property bool isSelected: craftMarketWindow.categoryFilter === modelData
                        implicitWidth: catLbl.implicitWidth + 12; implicitHeight: 24; radius: 3
                        color: isSelected ? "#324866" : "#1a2536"
                        border.width: 1; border.color: isSelected ? "#54cf82" : "#2f4057"
                        Label {
                            id: catLbl; anchors.centerIn: parent
                            text: modelData
                            color: parent.isSelected ? "#54cf82" : "#9eb0c7"
                            font.pixelSize: 10; font.bold: parent.isSelected
                        }
                        TapHandler {
                            onTapped: craftMarketWindow.categoryFilter = modelData
                        }
                        HoverHandler { cursorShape: Qt.PointingHandCursor }
                    }
                }
            }
        }

        // Lista de Recetas y Rentabilidad
        ListView {
            id: recipeListView
            Layout.fillWidth: true; Layout.fillHeight: true
            clip: true; spacing: 6
            model: {
                const all = backend.craftRecipes
                const res = []
                const sText = craftMarketWindow.searchText
                const exText = craftMarketWindow.excludeText

                for (let i = 0; i < all.length; ++i) {
                    const r = all[i]
                    if (craftMarketWindow.categoryFilter !== "TODOS" && r.categoryName !== craftMarketWindow.categoryFilter) continue
                    
                    // Exclusión de recursos/materiales
                    if (exText !== "") {
                        let containsExcluded = false
                        if (r.materials) {
                            for (let j = 0; j < r.materials.length; ++j) {
                                const mName = (r.materials[j].name || "").toLowerCase()
                                if (mName.includes(exText)) {
                                    containsExcluded = true
                                    break
                                }
                            }
                        }
                        if (containsExcluded) continue
                    }

                    // Búsqueda por texto
                    if (sText !== "") {
                        const nameMatch = (r.name || "").toLowerCase().includes(sText)
                        const origMatch = (r.originalName || "").toLowerCase().includes(sText)
                        let matMatch = false
                        if (r.materials) {
                            for (let j = 0; j < r.materials.length; ++j) {
                                if ((r.materials[j].name || "").toLowerCase().includes(sText)) {
                                    matMatch = true; break;
                                }
                            }
                        }
                        if (!nameMatch && !origMatch && !matMatch) continue
                    }

                    if (craftMarketWindow.onlyProfitable && Number(r.profit || 0) <= 0) continue

                    // Si hay presupuesto definido, podemos calcular los datos óptimos de presupuesto
                    if (craftMarketWindow.availableBudget > 0) {
                        const bSim = backend.simulateBatchCraft(r, 1, craftMarketWindow.availableBudget)
                        r.budgetAffordable = bSim.budgetAffordableCount
                        r.budgetProfit = bSim.budgetProfit
                        r.budgetTotalCost = bSim.budgetTotalCost
                        r.budgetRoi = bSim.budgetRoi
                        r.optimalWithBudget = bSim.optimalWithBudget
                    }

                    res.push(r)
                }

                // Ordenamiento
                const asc = craftMarketWindow.sortAscending
                const sortBy = craftMarketWindow.sortBy
                res.sort(function(a, b) {
                    let va = 0, vb = 0
                    if (sortBy === "profit") {
                        va = Number(a.profit || 0); vb = Number(b.profit || 0)
                    } else if (sortBy === "budgetProfit") {
                        va = Number(a.budgetProfit || 0); vb = Number(b.budgetProfit || 0)
                    } else if (sortBy === "roi") {
                        va = Number(a.roiPct || 0); vb = Number(b.roiPct || 0)
                    } else if (sortBy === "sellPrice") {
                        va = Number(a.sellPrice || 0); vb = Number(b.sellPrice || 0)
                    } else if (sortBy === "cost") {
                        va = Number(a.totalCost || 0); vb = Number(b.totalCost || 0)
                    } else if (sortBy === "name") {
                        const na = (a.name || "").toLowerCase()
                        const nb = (b.name || "").toLowerCase()
                        return asc ? na.localeCompare(nb) : nb.localeCompare(na)
                    }
                    return asc ? (va - vb) : (vb - va)
                })

                return res
            }

            delegate: Rectangle {
                id: recipeCard
                required property var modelData
                width: recipeListView.width
                implicitHeight: cardCol.implicitHeight + 14
                radius: 4
                color: "#182333"
                border.width: 1
                border.color: Number(modelData.profit || 0) > 0 ? "#3b6e51" : "#28374d"

                property bool expanded: false

                ColumnLayout {
                    id: cardCol
                    anchors.fill: parent; anchors.margins: 8; spacing: 4

                    // Fila Principal de la receta
                    RowLayout {
                        Layout.fillWidth: true; spacing: 8
                        Label {
                            text: "[" + (modelData.categoryName || "Crafteo") + "]"
                            color: "#8fa1b8"; font.pixelSize: 9; font.bold: true
                        }
                        Label {
                            text: modelData.name || "Receta"
                            color: "#ffffff"; font.pixelSize: 11; font.bold: true
                            Layout.fillWidth: true; elide: Text.ElideRight
                        }
                        // Badge de Costo
                        ColumnLayout {
                            spacing: 1
                            Label {
                                text: "Costo: " + (Number(modelData.totalCost || 0) > 0 ? (craftMarketWindow.formatKinah(modelData.totalCost) + " K") : "Sin cotizar")
                                color: "#e09090"; font.pixelSize: 9
                            }
                            Label {
                                text: "Venta: " + (Number(modelData.sellPrice || 0) > 0 ? (craftMarketWindow.formatKinah(modelData.sellPrice) + " K") : "Sin cotizar")
                                color: "#9ad9ec"; font.pixelSize: 9
                            }
                        }
                        // Badge de Beneficio
                        ColumnLayout {
                            spacing: 1
                            Rectangle {
                                implicitWidth: profitLbl.implicitWidth + 10; implicitHeight: 22; radius: 3
                                color: Number(modelData.profit || 0) > 0 ? "#1c3d2c" : Number(modelData.profit || 0) < 0 ? "#422126" : "#243142"
                                border.width: 1
                                border.color: Number(modelData.profit || 0) > 0 ? "#54cf82" : Number(modelData.profit || 0) < 0 ? "#ff8080" : "#455973"
                                Label {
                                    id: profitLbl; anchors.centerIn: parent
                                    text: {
                                        const p = Number(modelData.profit || 0)
                                        const roi = Number(modelData.roiPct || 0).toFixed(0)
                                        if (p > 0) return "+" + craftMarketWindow.formatKinah(p) + " K (+" + roi + "%)"
                                        if (p < 0) return craftMarketWindow.formatKinah(p) + " K (" + roi + "%)"
                                        return "0 K"
                                    }
                                    color: Number(modelData.profit || 0) > 0 ? "#54cf82" : Number(modelData.profit || 0) < 0 ? "#ff9e9e" : "#c0cbdc"
                                    font.pixelSize: 10; font.bold: true
                                }
                            }
                            // Si el usuario especificó un presupuesto, mostrar cuánto puede craftear y su ganancia total
                            Label {
                                visible: craftMarketWindow.availableBudget > 0
                                text: {
                                    const aff = Number(modelData.budgetAffordable || 0)
                                    const bProf = Number(modelData.budgetProfit || 0)
                                    if (aff <= 0) return "Presupuesto insuficiente"
                                    return "💰 " + aff + "x max (" + (bProf >= 0 ? "+" : "") + craftMarketWindow.formatKinah(bProf) + " K)"
                                }
                                color: Number(modelData.budgetProfit || 0) > 0 ? "#54cf82" : "#ff9e9e"
                                font.pixelSize: 8; font.bold: true
                            }
                        }
                        Label {
                            text: recipeCard.expanded ? "▲" : "▼"
                            color: "#8fa1b8"; font.pixelSize: 9
                        }
                    }

                    // Detalle Desplegable de Materiales y Simulador de Volumen
                    ColumnLayout {
                        visible: recipeCard.expanded
                        Layout.fillWidth: true; Layout.topMargin: 4; spacing: 6
                        Rectangle { Layout.fillWidth: true; height: 1; color: "#25344a" }
                        
                        Label {
                            text: "Materiales requeridos para fabricar (1 unidad):"
                            color: "#f1c778"; font.pixelSize: 9; font.bold: true
                        }
                        Repeater {
                            model: modelData.materials || []
                            delegate: RowLayout {
                                required property var modelData
                                Layout.fillWidth: true; spacing: 6
                                Label {
                                    text: "• " + modelData.count + "x " + modelData.name
                                    color: "#c0cbdc"; font.pixelSize: 9
                                    Layout.fillWidth: true; elide: Text.ElideRight
                                }
                                Label {
                                    text: {
                                        const p = Number(modelData.unitPrice || 0)
                                        const lots = Number(modelData.listings || 0)
                                        if (p <= 0) return "Precio no listado"
                                        return craftMarketWindow.formatKinah(p) + " K c/u" + (lots > 0 ? (" (" + lots + " lotes)") : "")
                                    }
                                    color: Number(modelData.unitPrice || 0) > 0 ? "#9ad9ec" : "#7e8ea6"
                                    font.pixelSize: 9
                                }
                            }
                        }

                        // Sección de Simulación de Volumen / Profundidad de Mercado
                        Rectangle {
                            Layout.fillWidth: true; radius: 4
                            color: "#131a26"; border.width: 1; border.color: "#2a3d54"
                            implicitHeight: simCol.implicitHeight + 14

                            property int batchCount: 10
                            property var simResult: recipeCard.expanded ? backend.simulateBatchCraft(recipeCard.modelData, batchCount, craftMarketWindow.availableBudget) : null

                            ColumnLayout {
                                id: simCol
                                anchors.fill: parent; anchors.margins: 8; spacing: 6

                                RowLayout {
                                    Layout.fillWidth: true; spacing: 6
                                    Label {
                                        text: "📊 Simulador de Volumen Acumulado (Profundidad de Mercado)"
                                        color: "#e8effc"; font.pixelSize: 10; font.bold: true
                                        Layout.fillWidth: true
                                    }
                                    Label {
                                        text: "Lote: " + parent.parent.batchCount + " crafteos"
                                        color: "#6cb2eb"; font.pixelSize: 10; font.bold: true
                                    }
                                }

                                // Si el usuario tiene un presupuesto fijado, mostrar tarjeta de recomendación y botón para sincronizar lote
                                Rectangle {
                                    visible: craftMarketWindow.availableBudget > 0
                                    Layout.fillWidth: true; implicitHeight: budgetBoxCol.implicitHeight + 10; radius: 4
                                    color: "#182a20"; border.width: 1; border.color: "#357049"
                                    ColumnLayout {
                                        id: budgetBoxCol
                                        anchors.fill: parent; anchors.margins: 6; spacing: 4
                                        RowLayout {
                                            Layout.fillWidth: true; spacing: 6
                                            Label {
                                                text: "🎯 ANÁLISIS CON TU KINAH (" + craftMarketWindow.formatKinah(craftMarketWindow.availableBudget) + " K DISPONIBLES):"
                                                color: "#f1c778"; font.pixelSize: 9; font.bold: true
                                                Layout.fillWidth: true
                                            }
                                            // Botón para ajustar el lote exactamente al límite de tu dinero
                                            Rectangle {
                                                property int optCount: parent.parent.parent.parent.simResult ? Number(parent.parent.parent.parent.simResult.optimalWithBudget || 0) : 0
                                                visible: optCount > 0
                                                implicitWidth: syncBtnLbl.implicitWidth + 10; implicitHeight: 20; radius: 3
                                                color: "#275936"; border.width: 1; border.color: "#54cf82"
                                                Label {
                                                    id: syncBtnLbl; anchors.centerIn: parent
                                                    text: "Fijar a " + parent.optCount + "x"
                                                    color: "#ffffff"; font.pixelSize: 9; font.bold: true
                                                }
                                                TapHandler {
                                                    onTapped: parent.parent.parent.parent.parent.batchCount = parent.optCount
                                                }
                                                HoverHandler { cursorShape: Qt.PointingHandCursor }
                                            }
                                        }
                                        RowLayout {
                                            Layout.fillWidth: true; spacing: 8
                                            Label {
                                                text: {
                                                    const res = parent.parent.parent.parent.simResult
                                                    if (!res) return ""
                                                    const aff = Number(res.budgetAffordableCount || 0)
                                                    const cost = Number(res.budgetTotalCost || 0)
                                                    const prof = Number(res.budgetProfit || 0)
                                                    const roi = Number(res.budgetRoi || 0).toFixed(0)
                                                    if (aff <= 0) return "Tu saldo actual no alcanza para comprar los ingredientes de 1 unidad."
                                                    return "Puedes fabricar hasta " + aff + " unidades · Costo total: " + craftMarketWindow.formatKinah(cost) + " K · Ganancia neta: " + (prof >= 0 ? "+" : "") + craftMarketWindow.formatKinah(prof) + " K (" + roi + "% ROI)"
                                                }
                                                color: "#e8effc"; font.pixelSize: 9
                                                Layout.fillWidth: true; wrapMode: Text.WordWrap
                                            }
                                        }
                                    }
                                }

                                // Botones rápidos de cantidad de crafteo
                                RowLayout {
                                    spacing: 4
                                    Label {
                                        text: "Multiplicador:"
                                        color: "#8fa1b8"; font.pixelSize: 9
                                    }
                                    Repeater {
                                        model: [1, 5, 10, 25, 50, 100]
                                        delegate: Rectangle {
                                            required property int modelData
                                            property bool isSelected: parent.parent.parent.batchCount === modelData
                                            implicitWidth: 32; implicitHeight: 20; radius: 3
                                            color: isSelected ? "#336699" : "#1e2c3d"
                                            border.width: 1; border.color: isSelected ? "#6cb2eb" : "#32445b"
                                            Label {
                                                anchors.centerIn: parent
                                                text: modelData + "x"
                                                color: parent.isSelected ? "#ffffff" : "#9eb0c7"
                                                font.pixelSize: 9; font.bold: parent.isSelected
                                            }
                                            TapHandler {
                                                onTapped: parent.parent.parent.batchCount = modelData
                                            }
                                            HoverHandler { cursorShape: Qt.PointingHandCursor }
                                        }
                                    }
                                }

                                // Resultados del cálculo acumulado
                                RowLayout {
                                    Layout.fillWidth: true; spacing: 8
                                    // Costo acumulado
                                    ColumnLayout {
                                        spacing: 1
                                        Label { text: "Costo Total Acumulado"; color: "#8fa1b8"; font.pixelSize: 8 }
                                        Label {
                                            text: parent.parent.parent.parent.simResult ? (craftMarketWindow.formatKinah(parent.parent.parent.parent.simResult.totalCost) + " K") : "-"
                                            color: "#ff9999"; font.pixelSize: 10; font.bold: true
                                        }
                                        Label {
                                            text: parent.parent.parent.parent.simResult ? ("Prom: " + craftMarketWindow.formatKinah(parent.parent.parent.parent.simResult.avgCostPerCraft) + " K/u") : ""
                                            color: "#b07b82"; font.pixelSize: 8
                                        }
                                    }
                                    // Ingreso neto acumulado (después del 10% broker tax)
                                    ColumnLayout {
                                        spacing: 1
                                        Label { text: "Venta Neta (-10% broker)"; color: "#8fa1b8"; font.pixelSize: 8 }
                                        Label {
                                            text: parent.parent.parent.parent.simResult ? (craftMarketWindow.formatKinah(parent.parent.parent.parent.simResult.totalNetRevenue) + " K") : "-"
                                            color: "#9ad9ec"; font.pixelSize: 10; font.bold: true
                                        }
                                        Label {
                                            text: parent.parent.parent.parent.simResult ? ("Prom: " + craftMarketWindow.formatKinah(parent.parent.parent.parent.simResult.avgNetRevenuePerCraft) + " K/u") : ""
                                            color: "#7292a8"; font.pixelSize: 8
                                        }
                                    }
                                    // Beneficio neto total
                                    ColumnLayout {
                                        Layout.fillWidth: true; spacing: 1
                                        Label { text: "Beneficio Neto Total"; color: "#8fa1b8"; font.pixelSize: 8 }
                                        Label {
                                            text: {
                                                const res = parent.parent.parent.parent.simResult
                                                if (!res) return "-"
                                                const p = Number(res.totalProfit || 0)
                                                const roi = Number(res.totalRoi || 0).toFixed(0)
                                                return (p > 0 ? "+" : "") + craftMarketWindow.formatKinah(p) + " K (" + (p > 0 ? "+" : "") + roi + "%)"
                                            }
                                            color: {
                                                const res = parent.parent.parent.parent.simResult
                                                if (!res) return "#c0cbdc"
                                                return Number(res.totalProfit || 0) > 0 ? "#54cf82" : "#ff8080"
                                            }
                                            font.pixelSize: 10; font.bold: true
                                        }
                                    }
                                }

                                // Indicador de Capacidad Máxima de Crafteo con Rentabilidad
                                Rectangle {
                                    Layout.fillWidth: true; implicitHeight: 22; radius: 3
                                    color: {
                                        const res = parent.parent.simResult
                                        if (!res) return "#1e2836"
                                        return Number(res.maxProfitableBatch || 0) > 0 ? "#183624" : "#3b1e22"
                                    }
                                    border.width: 1
                                    border.color: {
                                        const res = parent.parent.simResult
                                        if (!res) return "#2c3b4d"
                                        return Number(res.maxProfitableBatch || 0) > 0 ? "#2d6b45" : "#6e2930"
                                    }

                                    RowLayout {
                                        anchors.fill: parent; anchors.leftMargin: 6; anchors.rightMargin: 6; spacing: 4
                                        Label {
                                            text: {
                                                const res = parent.parent.parent.simResult
                                                if (!res) return "Calculando rentabilidad..."
                                                const maxB = Number(res.maxProfitableBatch || 0)
                                                if (maxB <= 0) return "⚠️ No rentable desde la 1ª unidad con los precios actuales."
                                                if (maxB >= 100) return "✨ Excelente liquidez: Rentable por más de 100 unidades continuas."
                                                return "⚡ Límite Seguro: Rentable hasta " + maxB + " unidades seguidas (después suben los materiales y genera pérdidas)."
                                            }
                                            color: {
                                                const res = parent.parent.parent.simResult
                                                if (!res) return "#8fa1b8"
                                                return Number(res.maxProfitableBatch || 0) > 0 ? "#6ee69e" : "#ff9e9e"
                                            }
                                            font.pixelSize: 9; font.bold: true
                                            Layout.fillWidth: true; elide: Text.ElideRight
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                TapHandler {
                    onTapped: recipeCard.expanded = !recipeCard.expanded
                }
            }
        }
    }
}
