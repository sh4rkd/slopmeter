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
    property bool onlyProfitable: false

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
            Layout.fillWidth: true; implicitHeight: 22; radius: 3
            color: "#1c2738"; border.width: 1; border.color: "#2a3d57"
            RowLayout {
                anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8
                Label {
                    text: backend.marketStatus
                    color: "#9ad9ec"; font.pixelSize: 10
                    Layout.fillWidth: true; elide: Text.ElideRight
                }
                Label {
                    text: "Impuesto estimado Broker: 7%"
                    color: "#7e8ea6"; font.pixelSize: 9
                }
            }
        }

        // Filtros y Búsqueda
        RowLayout {
            Layout.fillWidth: true; spacing: 8
            TextField {
                id: searchInput
                placeholderText: "Buscar receta o material..."
                implicitHeight: 28
                Layout.fillWidth: true
                palette.text: "#ffffff"
                palette.base: "#1a2433"
                onTextChanged: craftMarketWindow.searchText = text.toLowerCase()
            }
            CheckBox {
                text: "Solo Rentables (Beneficio > 0)"
                checked: craftMarketWindow.onlyProfitable
                palette.windowText: "#54cf82"
                onToggled: craftMarketWindow.onlyProfitable = checked
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
                for (let i = 0; i < all.length; ++i) {
                    const r = all[i]
                    if (craftMarketWindow.categoryFilter !== "TODOS" && r.categoryName !== craftMarketWindow.categoryFilter) continue
                    if (craftMarketWindow.searchText !== "") {
                        const nameMatch = (r.name || "").toLowerCase().includes(craftMarketWindow.searchText)
                        const origMatch = (r.originalName || "").toLowerCase().includes(craftMarketWindow.searchText)
                        let matMatch = false
                        if (r.materials) {
                            for (let j = 0; j < r.materials.length; ++j) {
                                if ((r.materials[j].name || "").toLowerCase().includes(craftMarketWindow.searchText)) {
                                    matMatch = true; break;
                                }
                            }
                        }
                        if (!nameMatch && !origMatch && !matMatch) continue
                    }
                    if (craftMarketWindow.onlyProfitable && Number(r.profit || 0) <= 0) continue
                    res.push(r)
                }
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
                                text: "Costo: " + (Number(modelData.totalCost || 0) > 0 ? Number(modelData.totalCost || 0).toLocaleString() + " K" : "Sin cotizar")
                                color: "#e09090"; font.pixelSize: 9
                            }
                            Label {
                                text: "Venta: " + (Number(modelData.sellPrice || 0) > 0 ? Number(modelData.sellPrice || 0).toLocaleString() + " K" : "Sin cotizar")
                                color: "#9ad9ec"; font.pixelSize: 9
                            }
                        }
                        // Badge de Beneficio
                        Rectangle {
                            implicitWidth: profitLbl.implicitWidth + 10; implicitHeight: 24; radius: 3
                            color: Number(modelData.profit || 0) > 0 ? "#1c3d2c" : Number(modelData.profit || 0) < 0 ? "#422126" : "#243142"
                            border.width: 1
                            border.color: Number(modelData.profit || 0) > 0 ? "#54cf82" : Number(modelData.profit || 0) < 0 ? "#ff8080" : "#455973"
                            Label {
                                id: profitLbl; anchors.centerIn: parent
                                text: {
                                    const p = Number(modelData.profit || 0)
                                    const roi = Number(modelData.roiPct || 0).toFixed(0)
                                    if (p > 0) return "+" + p.toLocaleString() + " K (+" + roi + "%)"
                                    if (p < 0) return p.toLocaleString() + " K (" + roi + "%)"
                                    return "0 K"
                                }
                                color: Number(modelData.profit || 0) > 0 ? "#54cf82" : Number(modelData.profit || 0) < 0 ? "#ff9e9e" : "#c0cbdc"
                                font.pixelSize: 10; font.bold: true
                            }
                        }
                        Label {
                            text: recipeCard.expanded ? "▲" : "▼"
                            color: "#8fa1b8"; font.pixelSize: 9
                        }
                    }

                    // Detalle Desplegable de Materiales
                    ColumnLayout {
                        visible: recipeCard.expanded
                        Layout.fillWidth: true; Layout.topMargin: 4; spacing: 2
                        Rectangle { Layout.fillWidth: true; height: 1; color: "#25344a" }
                        Label {
                            text: "Materiales requeridos para fabricar:"
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
                                    text: Number(modelData.unitPrice || 0) > 0 ? (Number(modelData.unitPrice).toLocaleString() + " K c/u = " + Number(modelData.totalCost || 0).toLocaleString() + " K") : "Precio no listado"
                                    color: Number(modelData.unitPrice || 0) > 0 ? "#9ad9ec" : "#7e8ea6"
                                    font.pixelSize: 9
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
