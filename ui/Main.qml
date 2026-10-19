import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
Window {
    id: meter
    visible: false; width: 800; height: 600; color: "transparent"
    title: "SLOPMETER"
    flags: Qt.FramelessWindowHint | Qt.WindowDoesNotAcceptFocus
    function openSettings() {
        settingsLoader.active = true
        if (settingsLoader.item) {settingsLoader.item.transientParent = null;settingsLoader.item.screen = meter.screen;settingsLoader.item.show();settingsLoader.item.raise();settingsLoader.item.requestActivate()}
    }
    Loader {id: settingsLoader; active: false; source: "Settings.qml"}
    function openDetails(actorId) {
        backend.selectPlayer(actorId)
        detailLoader.active = true
        if (detailLoader.item) {detailLoader.item.transientParent = null;detailLoader.item.screen = meter.screen;detailLoader.item.show();detailLoader.item.raise();detailLoader.item.requestActivate()}
    }
    Loader {id: detailLoader; active: false; source: "FightDetails.qml"}
    function clock(seconds) { const value = Math.max(0, Math.floor(Number(seconds || 0))); return Math.floor(value/60) + ":" + (value%60).toString().padStart(2,"0") }
    function number(value) { return Number(value || 0).toLocaleString(Qt.locale(), 'f', 0) }
    function partyDps() {
        let sum = 0
        const players = backend.displayPlayers || []
        for (let i = 0; i < players.length; ++i) sum += Number(players[i].dps || 0)
        return sum
    }
    readonly property real bossTimeLimitSeconds: 300
    function bossEtaKnown() {
        if (!backend.boss || !backend.boss.known) return false
        if (backend.boss.etaKnown !== undefined) return Boolean(backend.boss.etaKnown)
        return Number(backend.boss.hp || 0) === 0 || partyDps() > 0
    }
    function bossEtaSeconds() {
        if (!backend.boss || !backend.boss.known) return 0
        if (backend.boss.etaKnown !== undefined) return Math.max(0, Number(backend.boss.eta || 0))
        const hp = Number(backend.boss.hp || 0)
        if (hp <= 0) return 0
        const dps = partyDps()
        return dps > 0 ? hp / dps : 0
    }
    function bossTotalSeconds() {
        const elapsed = Math.max(0, Number(backend.duration || 0))
        if (backend.boss && backend.boss.totalEta !== undefined && bossEtaKnown()) return Math.max(elapsed, Number(backend.boss.totalEta || 0))
        return elapsed + bossEtaSeconds()
    }
    function bossTimeLeftSeconds() {
        if (backend.boss && backend.boss.timeLeft !== undefined) return Math.max(0, Number(backend.boss.timeLeft || 0))
        return Math.max(0, bossTimeLimitSeconds - Math.max(0, Number(backend.duration || 0)))
    }
    function etaColor(totalSeconds, known) {
        if (!known) return "#8e9eb5"
        const value = Number(totalSeconds || 0)
        if (value > 300) return "#ee6464"
        if (value >= 240) return "#f1c778"
        return "#54cf82"
    }
    function etaBackgroundColor(totalSeconds, known) {
        if (!known) return "#273348"
        const value = Number(totalSeconds || 0)
        if (value > 300) return "#3d252b"
        if (value >= 240) return "#3b3424"
        return "#21382e"
    }
    function compactNumber(value) {
        const rounded = Math.round(Number(value || 0))
        if (rounded >= 999950) return (rounded / 1000000).toLocaleString(Qt.locale(), 'f', 1) + "M"
        if (rounded >= 1000) return (rounded / 1000).toLocaleString(Qt.locale(), 'f', 1) + "K"
        return number(rounded)
    }
    function dpsText(value) {
        const amount = Number(value || 0)
        if (!backend.roundedDps) return number(amount)
        return compactNumber(amount)
    }
    function minimalistDamageText(value) {
        const amount = Number(value || 0)
        if (!backend.minimalistCompactDamage) return number(amount)
        return compactNumber(amount)
    }
    function scoreValueText(value) {
        const amount = Number(value || 0)
        if (backend.minimalistMode ? !backend.minimalistCompactDamage : !backend.roundedDps) return number(amount)
        return compactNumber(amount)
    }
    function bossHpValueText(value) {
        const amount = Number(value || 0)
        if (backend.minimalistMode && backend.minimalistCompactDamage) return compactNumber(amount)
        return number(amount)
    }
    function classColor(name) {
        // Match A2Tools' jobColorMap; Elementalist is named Spiritmaster there.
        const colors = {Gladiator: "#4FD1C5", Templar: "#5F8CFF", Ranger: "#41D98A", Assassin: "#7BE35A",
            Elementalist: "#E06BFF", Sorcerer: "#9A6BFF", Cleric: "#F2C15A", Chanter: "#FF9A3D", Brawler: "#E85D5D"}
        if (!colors[name]) return "#53677c"
        const tint = Qt.color(colors[name])
        return Qt.hsla(tint.hslHue, tint.hslSaturation * 0.45, tint.hslLightness * 0.78, tint.a)
    }
    function classTextColor(name) {
        const colors = {Gladiator: "#4FD1C5", Templar: "#5F8CFF", Ranger: "#41D98A", Assassin: "#7BE35A",
            Elementalist: "#E06BFF", Sorcerer: "#9A6BFF", Cleric: "#F2C15A", Chanter: "#FF9A3D", Brawler: "#E85D5D"}
        return colors[name] || "#d7e2f0"
    }
    Rectangle {
        id: panel
        objectName: "meterPanel"
        x: 40; y: 100
        readonly property int marginSize: backend.minimalistMode ? 8 : 14
        readonly property int rowHeight: backend.minimalistMode ? 22 : backend.barHeight
        width: Math.min(backend.barWidth + marginSize * 2, meter.width)
        height: Math.min(content.implicitHeight + marginSize * 2, meter.height)
        radius: 10; color: Qt.rgba(25/255,32/255,45/255,backend.backgroundOpacity/100); border.color: "#42516a"; border.width: backend.showBorder ? 1 : 0
        opacity: backend.overallOpacity / 100
        layer.enabled: opacity < 1
        ScrollView {
            anchors.fill: parent; anchors.margins: panel.marginSize
            clip: true; contentWidth: availableWidth
            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
            ScrollBar.vertical.policy: ScrollBar.AsNeeded
            ColumnLayout {
                id: content
                width: parent.width; spacing: backend.minimalistMode ? 4 : 8
                RowLayout {
                    Layout.fillWidth: true
                    Item {
                        Layout.fillWidth: true; implicitHeight: backend.minimalistMode ? 24 : 32
                        RowLayout {
                            anchors.fill: parent; spacing: 8
                            Label {text: "⠿  SLOPMETER"; color: "#9ad9ec"; font.bold: true; font.pixelSize: 11}
                            Label {
                                objectName: "headerCharacter"
                                text: backend.character || "Relog"; color: backend.character ? "#f2f5fa" : "#f1c778"
                                Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight; horizontalAlignment: Text.AlignRight; font.pixelSize: 11
                                ToolTip.visible: characterHover.hovered; ToolTip.text: backend.character || "Character not detected. Please relog or change zone"
                                HoverHandler {id: characterHover}
                            }
                        }
                        DragHandler {
                            target: panel
                            xAxis.minimum: 0; xAxis.maximum: Math.max(0,meter.width-panel.width)
                            yAxis.minimum: 0; yAxis.maximum: Math.max(0,meter.height-panel.height)
                            onActiveChanged: { if(active) backend.beginDrag(); else backend.endDrag() }
                            cursorShape: active ? Qt.ClosedHandCursor : Qt.OpenHandCursor
                        }
                    }
                    Button {
                        objectName: "radarToggleButton"
                        text: "◎"
                        implicitWidth: backend.minimalistMode ? 24 : 32; implicitHeight: backend.minimalistMode ? 24 : 32
                        palette.buttonText: backend.radarEnabled ? "#54cf82" : "#e6edf6"
                        onClicked: backend.radarEnabled = !backend.radarEnabled
                        ToolTip.visible: hovered; ToolTip.text: backend.radarEnabled ? "Radar detector ON (Cofres / Odyle / Recursos)" : "Toggle Radar detector (Cofres / Odyle / Recursos)"
                    }
                    Button { objectName: "settingsButton"; text: "⚙"; implicitWidth: backend.minimalistMode ? 24 : 32; implicitHeight: backend.minimalistMode ? 24 : 32; onClicked: meter.openSettings(); ToolTip.visible: hovered; ToolTip.text: "Meter settings" }
                    Button { objectName: "resetDamageButton"; text: "↻"; implicitWidth: backend.minimalistMode ? 24 : 32; implicitHeight: backend.minimalistMode ? 24 : 32; onClicked: backend.resetDamage(); ToolTip.visible: hovered; ToolTip.text: "Reset current fight damage"; Accessible.name: "Reset current fight damage" }
                    Button { objectName: "minimizeButton"; text: "−"; implicitWidth: backend.minimalistMode ? 24 : 32; implicitHeight: backend.minimalistMode ? 24 : 32; onClicked: backend.hide(); ToolTip.visible: hovered; ToolTip.text: "Hide; restore from the tray or launch SlopMeter again" }
                    Button { text: "×"; implicitWidth: backend.minimalistMode ? 24 : 32; implicitHeight: backend.minimalistMode ? 24 : 32; onClicked: backend.quit() }
                }
                ComboBox {
                    Layout.fillWidth: true
                    model: backend.fightChoices; textRole: "label"; valueRole: "id"
                    currentIndex: backend.selectedFightIndex
                    onActivated: backend.selectFight(currentValue)
                    visible: backend.fightChoices.length > 1
                    ToolTip.visible: hovered; ToolTip.text: "Last 10 fights · choose Current fight to return to live data"
                }
                ColumnLayout {
                    id: radarSection
                    objectName: "radarSection"
                    Layout.fillWidth: true
                    visible: backend.radarEnabled
                    spacing: backend.minimalistMode ? 2 : 4
                    property int selectedRadarId: backend.radarSelectedId
                    onSelectedRadarIdChanged: if (backend.radarSelectedId !== selectedRadarId) backend.radarSelectedId = selectedRadarId
                    property real radarRangeMeters: backend.radarRangeMeters
                    onRadarRangeMetersChanged: if (backend.radarRangeMeters !== radarRangeMeters) backend.radarRangeMeters = radarRangeMeters
                    property string activeSubmenu: backend.radarSubmenu
                    onActiveSubmenuChanged: if (backend.radarSubmenu !== activeSubmenu) backend.radarSubmenu = activeSubmenu
                    function selectedTarget() {
                        const rows = backend.displayRadar
                        if (!rows || rows.length === 0) return null
                        if (backend.radarSelectedId !== 0) {
                            for (let i = 0; i < rows.length; ++i) {
                                if (Number(rows[i].id) === backend.radarSelectedId) return rows[i]
                            }
                        }
                        return rows[0]
                    }
                    function horizArrow(dir) { return backend.radarHorizArrow(dir) }
                    function vertBadge(vDir, dz) { return backend.radarVertBadge(vDir, dz) }
                    function vertShort(vDir) { return backend.radarVertShort(vDir) }
                    function categoryColor(cat) { return backend.radarCategoryColor(cat) }
                    function categoryIcon(cat) { return backend.radarCategoryIcon(cat) }
                    function subItemsFor(catKey) { return backend.radarSubItemsFor(catKey) }
                    function hasTarget(t) { return Boolean(t && (t.name || t.id)) }
                    function toggleCategory(catKey, button) {
                        if (button === Qt.RightButton) {
                            backend.radarSubmenu = (backend.radarSubmenu === catKey) ? "" : catKey
                            return
                        }
                        if (catKey === "chest") backend.radarChests = !backend.radarChests
                        else if (catKey === "odyle") backend.radarOdyle = !backend.radarOdyle
                        else if (catKey === "gem") backend.radarGems = !backend.radarGems
                        else if (catKey === "ore") backend.radarOres = !backend.radarOres
                        else if (catKey === "herb") backend.radarHerbs = !backend.radarHerbs
                        else if (catKey === "wood") backend.radarWood = !backend.radarWood
                        else if (catKey === "other") backend.radarOther = !backend.radarOther
                        if (backend.radarSubItemsFor(catKey).length > 1) {
                            backend.radarSubmenu = catKey
                        }
                    }
                    RowLayout {
                        Layout.fillWidth: true; spacing: 3
                        Label {
                            text: "◎ MAPA"
                            color: "#54cf82"; font.pixelSize: 10; font.bold: true
                        }
                        Rectangle {
                            implicitWidth: rangeLbl.implicitWidth + 8; implicitHeight: 16; radius: 3
                            color: "#1f2b3d"; border.width: 1; border.color: "#384b66"
                            Label {
                                id: rangeLbl; anchors.centerIn: parent
                                text: Math.round(backend.radarRangeMeters) + "m"
                                color: "#9ad9ec"; font.pixelSize: 9; font.bold: true
                            }
                            TapHandler {
                                onTapped: {
                                    if (backend.radarRangeMeters <= 40) backend.radarRangeMeters = 70
                                    else if (backend.radarRangeMeters <= 70) backend.radarRangeMeters = 120
                                    else backend.radarRangeMeters = 40
                                }
                            }
                            HoverHandler { id: rangeHover; cursorShape: Qt.PointingHandCursor }
                            ToolTip.visible: rangeHover.hovered; ToolTip.text: "Cambiar zoom del radar (40m / 70m / 120m)"
                        }
                        Item { Layout.fillWidth: true }
                        Repeater {
                            model: [
                                { key: "chest", label: "Cubo", active: backend.radarChests, color: "#f2c15a" },
                                { key: "odyle", label: "Odyle", active: backend.radarOdyle, color: "#6be0ff" },
                                { key: "gem", label: "Gemas", active: backend.radarGems, color: "#e082ff" },
                                { key: "ore", label: "Mineral", active: backend.radarOres, color: "#ff9f5a" },
                                { key: "herb", label: "Hierbas", active: backend.radarHerbs, color: "#7be35a" },
                                { key: "wood", label: "Madera", active: backend.radarWood, color: "#c89b6e" },
                                { key: "other", label: "Otros", active: backend.radarOther, color: "#c0cbdc" }
                            ]
                            delegate: Rectangle {
                                required property var modelData
                                objectName: "radarFilter_" + modelData.key
                                implicitWidth: pillLabel.implicitWidth + 8
                                implicitHeight: 17; radius: 3
                                color: modelData.active ? Qt.rgba(40/255, 58/255, 78/255, 0.92) : "transparent"
                                border.width: 1
                                border.color: modelData.active ? modelData.color : "#3b495e"
                                Label {
                                    id: pillLabel
                                    anchors.centerIn: parent
                                    text: modelData.label
                                    font.pixelSize: 9; font.bold: modelData.active
                                    color: modelData.active ? modelData.color : "#7e8ea6"
                                }
                                TapHandler {
                                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                                    onTapped: function(eventPoint, button) {
                                        radarSection.toggleCategory(modelData.key, button)
                                    }
                                }
                                HoverHandler { cursorShape: Qt.PointingHandCursor }
                            }
                        }
                    }
                    Flow {
                        id: subFilterFlow
                        objectName: "radarSubFilterBar"
                        Layout.fillWidth: true
                        spacing: 3
                        visible: backend.radarSubmenu !== "" || backend.radarItemFilter !== ""
                        Rectangle {
                            implicitWidth: allSubLbl.implicitWidth + 8; implicitHeight: 16; radius: 3
                            color: backend.radarItemFilter === "" ? "#2b4257" : "transparent"
                            border.width: 1; border.color: backend.radarItemFilter === "" ? "#9ad9ec" : "#3b495e"
                            Label {
                                id: allSubLbl; anchors.centerIn: parent
                                text: "Todos"
                                font.pixelSize: 9; font.bold: backend.radarItemFilter === ""
                                color: backend.radarItemFilter === "" ? "#ffffff" : "#8e9eb5"
                            }
                            TapHandler { onTapped: { backend.radarItemFilter = ""; backend.radarSubmenu = "" } }
                            HoverHandler { cursorShape: Qt.PointingHandCursor }
                        }
                        Repeater {
                            model: backend.radarSubmenu !== "" ? backend.radarSubItemsFor(backend.radarSubmenu) : (backend.radarItemFilter !== "" ? [backend.radarItemFilter] : [])
                            delegate: Rectangle {
                                required property string modelData
                                implicitWidth: subItemLbl.implicitWidth + 8; implicitHeight: 16; radius: 3
                                property bool selected: backend.radarItemFilter === modelData
                                color: selected ? "#2b4257" : "#182230"
                                border.width: 1; border.color: selected ? "#54cf82" : "#35465c"
                                Label {
                                    id: subItemLbl; anchors.centerIn: parent
                                    text: modelData.replace(" Gemstone", "").replace(" Ore", "").replace(" Log", "")
                                    font.pixelSize: 9; font.bold: parent.selected
                                    color: parent.selected ? "#54cf82" : "#c0cbdc"
                                }
                                TapHandler {
                                    onTapped: backend.radarItemFilter = (backend.radarItemFilter === modelData ? "" : modelData)
                                }
                                HoverHandler { cursorShape: Qt.PointingHandCursor }
                            }
                        }
                    }
                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 8
                        Item {
                            id: radarMapContainer
                            objectName: "radarMapCanvas"
                            Layout.preferredWidth: backend.minimalistMode ? 112 : 128
                            Layout.preferredHeight: Layout.preferredWidth
                            property real radiusPx: width / 2
                            Rectangle {
                                anchors.fill: parent
                                radius: width / 2
                                color: backend.minimalistMode ? Qt.rgba(16/255, 24/255, 36/255, 0.72) : "#131c29"
                                border.width: 1.5
                                border.color: "#354963"
                            }
                            Rectangle {
                                anchors.centerIn: parent
                                width: parent.width * 0.66; height: width; radius: width / 2
                                color: "transparent"; border.width: 1; border.color: "#223147"
                            }
                            Rectangle {
                                anchors.centerIn: parent
                                width: parent.width * 0.33; height: width; radius: width / 2
                                color: "transparent"; border.width: 1; border.color: "#223147"
                            }
                            Rectangle {
                                anchors.centerIn: parent
                                width: parent.width - 6; height: 1
                                color: "#223147"
                            }
                            Rectangle {
                                anchors.centerIn: parent
                                width: 1; height: parent.height - 6
                                color: "#223147"
                            }
                            Label { anchors.top: parent.top; anchors.topMargin: 2; anchors.horizontalCenter: parent.horizontalCenter; text: "F"; color: "#54cf82"; font.pixelSize: 8; font.bold: true }
                            Label { anchors.bottom: parent.bottom; anchors.bottomMargin: 2; anchors.horizontalCenter: parent.horizontalCenter; text: "A"; color: "#7e8ea6"; font.pixelSize: 8 }
                            Label { anchors.left: parent.left; anchors.leftMargin: 4; anchors.verticalCenter: parent.verticalCenter; text: "I"; color: "#7e8ea6"; font.pixelSize: 8 }
                            Label { anchors.right: parent.right; anchors.rightMargin: 4; anchors.verticalCenter: parent.verticalCenter; text: "D"; color: "#7e8ea6"; font.pixelSize: 8 }
                            Rectangle {
                                id: playerCenterMarker
                                objectName: "radarPlayerCenter"
                                width: 8; height: 8; radius: 4
                                anchors.centerIn: parent
                                color: "#ffffff"
                                border.width: 1.5; border.color: "#54cf82"
                                z: 5
                            }
                            Repeater {
                                id: radarMapDots
                                model: backend.displayRadar
                                delegate: Item {
                                    required property var modelData
                                    required property int index
                                    objectName: "radarDot_" + index
                                    property real rx: Number(modelData.relX || 0)
                                    property real ry: Number(modelData.relY || 0)
                                    property real dist: Math.hypot(rx, ry)
                                    property real maxR: radarMapContainer.radiusPx - 10
                                    property real scaleFactor: dist > 0.001 ? Math.min(1.0, dist / backend.radarRangeMeters) * maxR / dist : 0.0
                                    property bool isSelected: {
                                        const sel = radarSection.selectedTarget()
                                        return radarSection.hasTarget(sel) && Number(sel.id) === Number(modelData.id)
                                    }
                                    x: radarMapContainer.radiusPx + rx * scaleFactor - width / 2
                                    y: radarMapContainer.radiusPx - ry * scaleFactor - height / 2
                                    width: isSelected ? 16 : 12
                                    height: isSelected ? 16 : 12
                                    z: isSelected ? 10 : (100 - index)
                                    Rectangle {
                                        anchors.centerIn: parent
                                        width: parent.isSelected ? 12 : 9
                                        height: width
                                        radius: width / 2
                                        color: backend.radarCategoryColor(modelData.category)
                                        border.width: parent.isSelected ? 2 : 1
                                        border.color: parent.isSelected ? "#ffffff" : "#101824"
                                    }
                                    Label {
                                        anchors.centerIn: parent
                                        text: backend.radarVertShort(modelData.vertDir)
                                        color: "#101824"
                                        font.pixelSize: 7; font.bold: true
                                    }
                                    TapHandler {
                                        onTapped: backend.radarSelectedId = Number(modelData.id)
                                    }
                                    HoverHandler { id: dotHover; cursorShape: Qt.PointingHandCursor }
                                    ToolTip.visible: dotHover.hovered
                                    ToolTip.text: (modelData.name || "Resource") + " · " + Number(modelData.distance || 0).toFixed(0) + "m · " + backend.radarVertBadge(modelData.vertDir, modelData.zDelta)
                                }
                            }
                        }
                        ColumnLayout {
                            Layout.fillWidth: true
                            Layout.alignment: Qt.AlignVCenter
                            spacing: 4
                            Rectangle {
                                id: selectedRadarCard
                                objectName: "selectedRadarCard"
                                visible: radarSection.hasTarget(radarSection.selectedTarget())
                                Layout.fillWidth: true
                                implicitHeight: selectedCol.implicitHeight + 10
                                radius: 4
                                color: backend.minimalistMode ? Qt.rgba(25/255, 36/255, 52/255, 0.85) : "#1f2b3d"
                                border.width: 1
                                border.color: radarSection.hasTarget(radarSection.selectedTarget()) ? backend.radarCategoryColor(radarSection.selectedTarget().category) : "#314259"
                                ColumnLayout {
                                    id: selectedCol
                                    anchors.fill: parent; anchors.margins: 5; spacing: 2
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 4
                                        Label {
                                            objectName: "radarSelectedName"
                                            text: {
                                                const t = radarSection.selectedTarget()
                                                return radarSection.hasTarget(t) ? (backend.radarCategoryIcon(t.category) + " " + (t.name || "Object")) : ""
                                            }
                                            color: {
                                                const t = radarSection.selectedTarget()
                                                return radarSection.hasTarget(t) ? backend.radarCategoryColor(t.category) : "#ffffff"
                                            }
                                            font.pixelSize: 11; font.bold: true
                                            Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                                        }
                                        Label {
                                            objectName: "radarSelectedDistance"
                                            text: {
                                                const t = radarSection.selectedTarget()
                                                return radarSection.hasTarget(t) ? (Number(t.distance || 0).toFixed(0) + "m") : ""
                                            }
                                            color: "#ffffff"; font.pixelSize: 11; font.bold: true
                                        }
                                    }
                                    Rectangle {
                                        Layout.fillWidth: true; implicitHeight: 20; radius: 3
                                        color: {
                                            const t = radarSection.selectedTarget()
                                            if (!radarSection.hasTarget(t)) return "#182230"
                                            return t.vertDir === "Arriba" ? "#1c3b4d" : t.vertDir === "Abajo" ? "#422332" : "#223042"
                                        }
                                        RowLayout {
                                            anchors.fill: parent; anchors.leftMargin: 5; anchors.rightMargin: 5; spacing: 4
                                            Label {
                                                objectName: "radarSelectedVert"
                                                text: {
                                                    const t = radarSection.selectedTarget()
                                                    return radarSection.hasTarget(t) ? backend.radarVertBadge(t.vertDir, t.zDelta) : ""
                                                }
                                                color: {
                                                    const t = radarSection.selectedTarget()
                                                    if (!radarSection.hasTarget(t)) return "#8e9eb5"
                                                    return t.vertDir === "Arriba" ? "#6be0ff" : t.vertDir === "Abajo" ? "#ff9e9e" : "#54cf82"
                                                }
                                                font.pixelSize: 10; font.bold: true
                                                Layout.fillWidth: true; elide: Text.ElideRight
                                            }
                                            Label {
                                                objectName: "radarSelectedHoriz"
                                                text: {
                                                    const t = radarSection.selectedTarget()
                                                    return radarSection.hasTarget(t) ? backend.radarHorizArrow(t.horizDir) : ""
                                                }
                                                color: "#c0cbdc"; font.pixelSize: 9; font.bold: true
                                            }
                                        }
                                    }
                                }
                            }
                            Repeater {
                                id: radarQuickList
                                model: Math.min(3, backend.displayRadar.length)
                                delegate: Rectangle {
                                    required property int index
                                    property var itemData: backend.displayRadar[index]
                                    property bool isCurrent: {
                                        const sel = radarSection.selectedTarget()
                                        return radarSection.hasTarget(sel) && itemData && Number(sel.id) === Number(itemData.id)
                                    }
                                    objectName: "radarRow_" + index
                                    Layout.fillWidth: true; implicitHeight: 18; radius: 3
                                    color: isCurrent ? "#283952" : (backend.minimalistMode ? "transparent" : "#182230")
                                    border.width: isCurrent ? 1 : 0
                                    border.color: itemData ? backend.radarCategoryColor(itemData.category) : "transparent"
                                    RowLayout {
                                        anchors.fill: parent; anchors.leftMargin: 4; anchors.rightMargin: 4; spacing: 4
                                        Label {
                                            text: itemData ? (backend.radarCategoryIcon(itemData.category) + " " + (itemData.name || "Object")) : ""
                                            color: itemData ? backend.radarCategoryColor(itemData.category) : "#c0cbdc"
                                            font.pixelSize: 9; font.bold: parent.parent.isCurrent
                                            Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                                        }
                                        Label {
                                            text: itemData ? (backend.radarVertShort(itemData.vertDir) + " " + Number(itemData.distance || 0).toFixed(0) + "m") : ""
                                            color: "#e6edf6"; font.pixelSize: 9; font.bold: true
                                        }
                                    }
                                    TapHandler {
                                        onTapped: if (itemData) backend.radarSelectedId = Number(itemData.id)
                                    }
                                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                                }
                            }
                            Label {
                                objectName: "radarEmptyLabel"
                                visible: backend.displayRadar.length === 0
                                Layout.fillWidth: true
                                text: "Escaneando mapa…\nsin objetivos en rango"
                                color: "#7e8ea6"; font.pixelSize: 10; horizontalAlignment: Text.AlignHCenter
                            }
                        }
                    }
                }
                ColumnLayout {
                    Layout.fillWidth: true; visible: !!backend.boss.entity && !backend.isPvP; spacing: backend.minimalistMode ? 1 : 3
                    Rectangle {
                        objectName: "targetHealthBar"
                        Layout.fillWidth: true; implicitHeight: backend.minimalistMode ? 18 : 26; radius: 4
                        color: backend.minimalistMode ? "transparent" : "#423139"
                        Rectangle {
                            visible: !backend.minimalistMode
                            width: parent.width * (backend.boss.maxKnown ? backend.boss.percent / 100 : 0)
                            height: parent.height; radius: 4; color: "#a44b57"
                            Behavior on width { NumberAnimation { duration: 120 } }
                        }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 6; spacing: 6
                            Label {text: backend.boss.name || "Target"; color: backend.minimalistMode ? "#ff9e9e" : "#fff1eb"; font.pixelSize: 11; font.bold: true; Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight}
                            Label {
                                color: "#fff1eb"; font.pixelSize: 11; Layout.maximumWidth: parent.width * 0.65; elide: Text.ElideRight
                                text: !backend.boss.known ? "HP unknown" : backend.boss.maxKnown ? meter.bossHpValueText(backend.boss.hp) + " / " + meter.bossHpValueText(backend.boss.max) + " · " + Number(backend.boss.percent).toFixed(1) + "%" : meter.bossHpValueText(backend.boss.hp) + " HP"
                            }
                        }
                        HoverHandler {id: targetHover}
                        ToolTip.visible: targetHover.hovered
                        ToolTip.text: (backend.boss.name || "Target") + (!backend.boss.known ? " · Waiting for captured HP" : !backend.boss.maxKnown ? " · Maximum HP not detected" : "")
                    }
                    Rectangle {
                        id: targetKillTimeBar
                        objectName: "targetKillTimeBar"
                        property bool etaReady: meter.bossEtaKnown()
                        property real elapsedSeconds: Math.max(0, Number(backend.duration || 0))
                        property real etaSeconds: meter.bossEtaSeconds()
                        property real totalSeconds: meter.bossTotalSeconds()
                        property real timeLeftSeconds: meter.bossTimeLeftSeconds()
                        property bool timeExpired: elapsedSeconds >= meter.bossTimeLimitSeconds
                        property bool colorReady: etaReady || timeExpired
                        property real effectiveTotalSeconds: etaReady ? totalSeconds : elapsedSeconds
                        property color statusColor: meter.etaColor(effectiveTotalSeconds, colorReady)
                        Layout.fillWidth: true; implicitHeight: backend.minimalistMode ? 30 : 38; radius: 4
                        color: backend.minimalistMode ? "transparent" : meter.etaBackgroundColor(effectiveTotalSeconds, colorReady)
                        border.width: backend.minimalistMode ? 0 : 1; border.color: colorReady ? Qt.rgba(statusColor.r, statusColor.g, statusColor.b, 0.55) : "#38465c"
                        ColumnLayout {
                            anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 6; anchors.topMargin: backend.minimalistMode ? 1 : 3; anchors.bottomMargin: backend.minimalistMode ? 1 : 3; spacing: 1
                            RowLayout {
                                Layout.fillWidth: true; spacing: 6
                                Rectangle {
                                    objectName: "targetKillTimeIndicator"
                                    Layout.preferredWidth: 7; Layout.preferredHeight: 7; radius: 3.5
                                    color: targetKillTimeBar.statusColor
                                }
                                Label {
                                    objectName: "targetPartyDpsLabel"
                                    text: "Party DPS · " + meter.dpsText(meter.partyDps())
                                    color: "#e6edf6"; font.pixelSize: 10
                                    Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                                }
                                Label {
                                    objectName: "targetWipeCountdownLabel"
                                    color: targetKillTimeBar.timeLeftSeconds <= 0 ? "#ee6464" : targetKillTimeBar.timeLeftSeconds <= 60 ? "#f1c778" : "#c7d4e6"
                                    font.pixelSize: 10; font.bold: true
                                    text: targetKillTimeBar.timeLeftSeconds <= 0 && Number(backend.boss.hp || 0) > 0 ? "Wipe (0:00 left)" : "Wipe left " + meter.clock(targetKillTimeBar.timeLeftSeconds)
                                }
                            }
                            RowLayout {
                                Layout.fillWidth: true; spacing: 6
                                Label {
                                    objectName: "targetElapsedTimeLabel"
                                    text: "Elapsed " + meter.clock(targetKillTimeBar.elapsedSeconds) + " / 5:00"
                                    color: "#b8c7dc"; font.pixelSize: 10
                                    Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                                }
                                Label {
                                    objectName: "targetKillTimeLabel"
                                    color: targetKillTimeBar.statusColor; font.pixelSize: 11; font.bold: true
                                    text: !backend.boss.known ? "HP unknown" : !targetKillTimeBar.etaReady ? "Waiting for DPS…" : Number(backend.boss.hp || 0) <= 0 ? "Defeated (" + meter.clock(targetKillTimeBar.elapsedSeconds) + ")" : "Est. ~" + meter.clock(targetKillTimeBar.etaSeconds) + " (tot ~" + meter.clock(targetKillTimeBar.totalSeconds) + ")"
                                }
                            }
                        }
                        HoverHandler {id: etaHover}
                        ToolTip.visible: etaHover.hovered
                        ToolTip.text: "5:00 boss wipe limit · Elapsed " + meter.clock(elapsedSeconds) + " · Time left " + meter.clock(timeLeftSeconds) + (etaReady ? " · Est. remaining ~" + meter.clock(etaSeconds) + " · Projected total ~" + meter.clock(totalSeconds) : "") + " (projected total < 4:00 green · 4:00–5:00 yellow · > 5:00 red)"
                    }
                }
                ColumnLayout {
                    id: pvpDuelSection
                    objectName: "pvpDuelHeader"
                    Layout.fillWidth: true; visible: backend.isPvP; spacing: backend.minimalistMode ? 1 : 3
                    property real dealtDmg: Number(backend.pvpSummary.damageDealt || 0)
                    property real takenDmg: Number(backend.pvpSummary.damageTaken || 0)
                    property real dealtDps: Number(backend.pvpSummary.dealtDps || 0)
                    property real takenDps: Number(backend.pvpSummary.takenDps || 0)
                    property real dealtRatio: backend.pvpSummary.dealtShare !== undefined ? Number(backend.pvpSummary.dealtShare) : 50
                    property string oppName: backend.pvpSummary.opponentName || "Opponent"
                    property string outcome: backend.pvpSummary.outcome || ""
                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: backend.minimalistMode ? 18 : 24; radius: 4
                        color: backend.minimalistMode ? "transparent" : "#3b1f2b"
                        border.width: backend.minimalistMode ? 0 : 1
                        border.color: pvpDuelSection.outcome === "Victory" ? "#54cf82" : pvpDuelSection.outcome === "Defeat" ? "#ee6464" : "#803f52"
                        Rectangle {
                            visible: !backend.minimalistMode
                            width: parent.width * Math.max(0, Math.min(100, pvpDuelSection.dealtRatio)) / 100
                            height: parent.height; radius: 4; color: "#2b5b6c"
                            Behavior on width { NumberAnimation { duration: 120 } }
                        }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 6; spacing: 6
                            Label {
                                objectName: "pvpHeaderTitle"
                                text: "⚔ PvP · vs " + pvpDuelSection.oppName + (pvpDuelSection.outcome ? " · " + pvpDuelSection.outcome.toUpperCase() : "")
                                color: pvpDuelSection.outcome === "Victory" ? "#54cf82" : pvpDuelSection.outcome === "Defeat" ? "#ff8080" : "#ffd6de"
                                font.pixelSize: 11; font.bold: true; Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                            }
                            Label {
                                visible: !!backend.boss.entity && backend.boss.known
                                color: "#fff1eb"; font.pixelSize: 10
                                text: backend.boss.maxKnown ? meter.bossHpValueText(backend.boss.hp) + " HP (" + Number(backend.boss.percent).toFixed(0) + "%)" : meter.bossHpValueText(backend.boss.hp) + " HP"
                            }
                        }
                    }
                    Rectangle {
                        objectName: "pvpExchangeBar"
                        Layout.fillWidth: true; implicitHeight: backend.minimalistMode ? 18 : 22; radius: 4
                        color: backend.minimalistMode ? "transparent" : "#222b3c"
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 6; spacing: 6
                            Label {
                                objectName: "pvpDealtSummaryLabel"
                                text: "Dealt " + (backend.minimalistMode ? meter.minimalistDamageText(pvpDuelSection.dealtDmg) : meter.compactNumber(pvpDuelSection.dealtDmg)) + " (" + meter.dpsText(pvpDuelSection.dealtDps) + "/s)"
                                color: "#9ad9ec"; font.pixelSize: 10; font.bold: true
                                Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                            }
                            Label {
                                text: Math.round(pvpDuelSection.dealtRatio) + "% / " + Math.round(100 - pvpDuelSection.dealtRatio) + "%"
                                color: pvpDuelSection.dealtDmg >= pvpDuelSection.takenDmg ? "#54cf82" : "#ff8e8e"
                                font.pixelSize: 10; font.bold: true
                            }
                            Label {
                                objectName: "pvpTakenSummaryLabel"
                                text: "Taken " + (backend.minimalistMode ? meter.minimalistDamageText(pvpDuelSection.takenDmg) : meter.compactNumber(pvpDuelSection.takenDmg)) + " (" + meter.dpsText(pvpDuelSection.takenDps) + "/s)"
                                color: "#ff9e9e"; font.pixelSize: 10; font.bold: true
                                horizontalAlignment: Text.AlignRight; Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight
                            }
                        }
                        HoverHandler {id: pvpHover}
                        ToolTip.visible: pvpHover.hovered
                        ToolTip.text: "Damage dealt: " + meter.number(pvpDuelSection.dealtDmg) + " (" + meter.number(pvpDuelSection.dealtDps) + " DPS) · Damage taken: " + meter.number(pvpDuelSection.takenDmg) + " (" + meter.number(pvpDuelSection.takenDps) + " DPS)"
                    }
                }
                Label {
                    objectName: "combatStatus"
                    Layout.fillWidth: true; horizontalAlignment: Text.AlignRight; elide: Text.ElideRight; color: backend.isPvP ? "#f2a6b8" : "#8e9eb5"; font.pixelSize: 11
                    text: (backend.sceneName ? backend.sceneName + " · " : "") + (backend.testMode ? "Test mode · " + meter.clock(backend.duration) : backend.detailFight.active ? (backend.isPvP ? "PvP · " : "Combat · ") + meter.clock(backend.duration) : (backend.displayPlayers.length || backend.displayEnemies.length) ? (backend.isPvP ? "PvP Finished · " : "Finished · ") + meter.clock(backend.duration) : "Waiting for combat")
                    ToolTip.visible: combatHover.hovered; ToolTip.text: backend.encounter
                    HoverHandler {id: combatHover}
                }
                ListView {
                    id: bars; objectName: "playerBars"
                    Layout.fillWidth: true
                    implicitHeight: count ? count * panel.rowHeight + Math.max(0,count-1) * backend.barSpacing : (backend.minimalistMode ? 24 : 36)
                    Layout.preferredHeight: implicitHeight
                    interactive: false; spacing: backend.barSpacing
                    model: actorModel
                    delegate: Rectangle {
                        id: bar
                        required property int index
                        property color classTint: meter.classColor(actorClass)
                        property color textColor: meter.classTextColor(actorClass)
                        required property int actorId
                        required property string actorName
                        required property string actorClass
                        required property real gearScore
                        required property real combatPower
                        required property real damageTaken
                        required property real criticalRate
                        required property bool criticalKnown
                        required property real maxHit
                        property bool showGear: backend.scoreDisplay === 1 || backend.scoreDisplay === 3
                        property bool showPower: backend.scoreDisplay === 1 || backend.scoreDisplay === 2
                        property string gearText: (showGear && gearScore > 0 ? "GS " + meter.scoreValueText(gearScore) : "") + (showGear && gearScore > 0 && showPower && combatPower > 0 ? " · " : "") + (showPower && combatPower > 0 ? "CP " + meter.scoreValueText(combatPower) : "")
                        property string pvpMetaText: backend.isPvP ? ((criticalKnown ? "Crit " + criticalRate.toFixed(0) + "%" : "") + (criticalKnown && maxHit > 0 ? " · " : "") + (maxHit > 0 ? "Max " + meter.scoreValueText(maxHit) : "") + ((criticalKnown || maxHit > 0) && damageTaken > 0 ? " · " : "") + (damageTaken > 0 ? "In " + meter.scoreValueText(damageTaken) : "")) : ""
                        property string subLabelText: backend.isPvP && pvpMetaText.length > 0 ? pvpMetaText : gearText
                        required property real damage
                        required property real dps
                        required property real share
                        required property real fill
                        TapHandler {onTapped: meter.openDetails(bar.actorId)}
                        HoverHandler {cursorShape: Qt.PointingHandCursor}
                        width: ListView.view.width; height: panel.rowHeight; radius: 4
                        color: backend.minimalistMode ? "transparent" : "#273348"
                        Rectangle {
                            id: damageFill
                            objectName: "damageFill"
                            visible: !backend.minimalistMode
                            width: parent.width * parent.fill / 100; height: parent.height; radius: 4; color: bar.classTint
                            gradient: backend.barStyle === 1 ? softGradient : backend.barStyle === 3 ? glossyGradient : null
                            Gradient {id: softGradient; GradientStop {position: 0; color: Qt.lighter(bar.classTint,1.18)} GradientStop {position: 1; color: Qt.darker(bar.classTint,1.18)}}
                            Gradient {id: glossyGradient; GradientStop {position: 0; color: Qt.lighter(bar.classTint,1.32)} GradientStop {position: 0.48; color: Qt.lighter(bar.classTint,1.06)} GradientStop {position: 0.50; color: bar.classTint} GradientStop {position: 1; color: Qt.darker(bar.classTint,1.22)}}
                            Rectangle {visible: backend.barStyle === 2; anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; height: 8; radius: 4; gradient: Gradient {GradientStop {position: 0; color: "#65000000"} GradientStop {position: 1; color: "transparent"}}}
                            Rectangle {visible: backend.barStyle === 2; anchors.fill: parent; radius: 4; color: "transparent"; border.width: 1; border.color: "#45000000"}
                            Behavior on width { NumberAnimation { duration: 120 } }
                        }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 8; spacing: backend.minimalistMode ? 5 : 7
                            Label {
                                objectName: "minimalistRank"
                                visible: backend.minimalistMode
                                text: (bar.index + 1) + "."
                                color: "#8e9eb5"; font.pixelSize: 11; font.bold: true
                                Layout.preferredWidth: 16; horizontalAlignment: Text.AlignRight
                            }
                            Image {
                                source: backend.classIcon(bar.actorClass)
                                sourceSize.width: 32; sourceSize.height: 32
                                Layout.preferredWidth: backend.minimalistMode ? 16 : Math.min(26,backend.barHeight-6); Layout.preferredHeight: Layout.preferredWidth
                                ToolTip.visible: iconHover.hovered; ToolTip.text: bar.actorClass || "Class not detected"
                                HoverHandler { id: iconHover }
                            }
                            ColumnLayout {
                                Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: 0
                                Label {
                                    objectName: "actorNameLabel"
                                    text: bar.actorName + (backend.minimalistMode && bar.gearText.length > 0 ? " (" + bar.gearText + ")" : "")
                                    color: backend.minimalistMode ? bar.textColor : "#ffffff"
                                    font.bold: true; font.pixelSize: backend.minimalistMode ? 12 : 13; elide: Text.ElideRight
                                    Layout.fillWidth: true; Layout.minimumWidth: 0
                                    HoverHandler {id: nameHover}
                                    ToolTip.visible: nameHover.hovered && ((backend.minimalistMode || backend.barHeight < 38) && bar.subLabelText.length > 0)
                                    ToolTip.text: bar.subLabelText
                                }
                                Label {
                                    visible: !backend.minimalistMode && backend.barHeight >= 38 && bar.subLabelText.length > 0
                                    text: bar.subLabelText; color: "#e6edf6"; font.pixelSize: 11; elide: Text.ElideRight
                                    Layout.fillWidth: true; Layout.minimumWidth: 0
                                }
                            }
                            Label {
                                objectName: "compactShare"
                                visible: !backend.minimalistMode && backend.showDetails && backend.barHeight < 38
                                text: bar.share.toFixed(1) + "%"; color: "#ffffff"; font.pixelSize: 11
                                Layout.preferredWidth: Math.max(40,bar.width * 0.15); horizontalAlignment: Text.AlignHCenter
                            }
                            Label {
                                objectName: "minimalistStats"
                                visible: backend.minimalistMode
                                color: bar.textColor; font.pixelSize: 12; font.bold: true
                                horizontalAlignment: Text.AlignRight
                                text: (backend.showTotalDamage ? meter.minimalistDamageText(bar.damage) + " · " : "") + meter.dpsText(bar.dps) + " DPS" + (backend.isPvP && bar.criticalKnown ? " · " + bar.criticalRate.toFixed(0) + "%C" : (!backend.isPvP && backend.showDetails ? " (" + bar.share.toFixed(1) + "%)" : "")) + (backend.isPvP && bar.damageTaken > 0 ? " · -" + meter.minimalistDamageText(bar.damageTaken) : "")
                            }
                            ColumnLayout {
                                visible: !backend.minimalistMode
                                Layout.preferredWidth: backend.barHeight < 38 ? Math.max(80,bar.width / 2 - Math.max(40,bar.width * 0.15) / 2 - 8) : implicitWidth
                                spacing: 0
                                Label {
                                    text: meter.dpsText(bar.dps) + " DPS"; color: "#ffffff"; font.bold: true; Layout.alignment: Qt.AlignRight
                                    HoverHandler {id: damageHover}
                                    ToolTip.visible: damageHover.hovered && backend.showTotalDamage && backend.barHeight < 38
                                    ToolTip.text: meter.number(bar.damage) + " total damage" + (backend.isPvP && bar.damageTaken > 0 ? " · " + meter.number(bar.damageTaken) + " taken" : "")
                                }
                                Label { visible: (backend.showTotalDamage || backend.showDetails) && backend.barHeight >= 38; text: (backend.showTotalDamage ? meter.number(bar.damage) + " dmg" : "") + (backend.showTotalDamage && backend.showDetails ? " · " : "") + (backend.showDetails ? bar.share.toFixed(1) + "%" : ""); color: "#e6edf6"; font.pixelSize: 11; Layout.alignment: Qt.AlignRight }
                            }
                        }
                    }
                    Label { anchors.centerIn: parent; visible: bars.count === 0; text: "Waiting for your party's damage…"; color: "#8e9eb5"; font.pixelSize: 12 }
                }
                ColumnLayout {
                    id: enemySection
                    objectName: "enemySection"
                    Layout.fillWidth: true
                    visible: backend.isPvP && enemyBars.count > 0
                    spacing: backend.minimalistMode ? 2 : 4
                    RowLayout {
                        Layout.fillWidth: true; spacing: 6
                        Rectangle { Layout.fillWidth: true; implicitHeight: 1; color: "#633240" }
                        Label {
                            objectName: "enemyDividerLabel"
                            text: "RIVALS · INCOMING DAMAGE"
                            color: "#ff9e9e"; font.pixelSize: 10; font.bold: true
                        }
                        Rectangle { Layout.fillWidth: true; implicitHeight: 1; color: "#633240" }
                    }
                    ListView {
                        id: enemyBars; objectName: "enemyBars"
                        Layout.fillWidth: true
                        implicitHeight: count * panel.rowHeight + Math.max(0, count - 1) * backend.barSpacing
                        Layout.preferredHeight: implicitHeight
                        interactive: false; spacing: backend.barSpacing
                        model: enemyModel
                        delegate: Rectangle {
                            id: enemyBar
                            required property int index
                            property color classTint: meter.classColor(actorClass)
                            property color textColor: meter.classTextColor(actorClass)
                            required property int actorId
                            required property string actorName
                            required property string actorClass
                            required property real damage
                            required property real damageTaken
                            required property real dps
                            required property real criticalRate
                            required property bool criticalKnown
                            required property real maxHit
                            required property real share
                            required property real fill
                            property string rivalMeta: (criticalKnown ? "Crit " + criticalRate.toFixed(0) + "%" : "") + (criticalKnown && maxHit > 0 ? " · " : "") + (maxHit > 0 ? "Max " + meter.scoreValueText(maxHit) : "") + ((criticalKnown || maxHit > 0) && damageTaken > 0 ? " · " : "") + (damageTaken > 0 ? "Taken " + meter.scoreValueText(damageTaken) : "")
                            TapHandler {onTapped: meter.openDetails(enemyBar.actorId)}
                            HoverHandler {cursorShape: Qt.PointingHandCursor}
                            width: ListView.view.width; height: panel.rowHeight; radius: 4
                            color: backend.minimalistMode ? "transparent" : "#38222c"
                            border.width: backend.minimalistMode ? 0 : 1
                            border.color: "#6e3546"
                            Rectangle {
                                visible: !backend.minimalistMode
                                width: parent.width * parent.fill / 100; height: parent.height; radius: 4
                                color: actorClass ? Qt.tint(enemyBar.classTint, "#40c93b4b") : "#8a3847"
                                Behavior on width { NumberAnimation { duration: 120 } }
                            }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: backend.minimalistMode ? 2 : 6; anchors.rightMargin: backend.minimalistMode ? 2 : 8; spacing: backend.minimalistMode ? 5 : 7
                                Label {
                                    visible: backend.minimalistMode
                                    text: "⚔"
                                    color: "#ff8080"; font.pixelSize: 11; font.bold: true
                                    Layout.preferredWidth: 16; horizontalAlignment: Text.AlignRight
                                }
                                Image {
                                    source: backend.classIcon(enemyBar.actorClass)
                                    sourceSize.width: 32; sourceSize.height: 32
                                    Layout.preferredWidth: backend.minimalistMode ? 16 : Math.min(26, backend.barHeight - 6); Layout.preferredHeight: Layout.preferredWidth
                                    ToolTip.visible: enemyIconHover.hovered; ToolTip.text: (enemyBar.actorClass || "Unknown class") + " · Rival (click for skill breakdown & timeline)"
                                    HoverHandler { id: enemyIconHover }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: 0
                                    Label {
                                        objectName: "enemyNameLabel"
                                        text: enemyBar.actorName
                                        color: backend.minimalistMode ? (enemyBar.actorClass ? enemyBar.textColor : "#ff9e9e") : "#ffdbe2"
                                        font.bold: true; font.pixelSize: backend.minimalistMode ? 12 : 13; elide: Text.ElideRight
                                        Layout.fillWidth: true; Layout.minimumWidth: 0
                                        HoverHandler { id: enemyNameHover }
                                        ToolTip.visible: enemyNameHover.hovered && enemyBar.rivalMeta.length > 0
                                        ToolTip.text: enemyBar.rivalMeta
                                    }
                                    Label {
                                        visible: !backend.minimalistMode && backend.barHeight >= 38 && enemyBar.rivalMeta.length > 0
                                        text: enemyBar.rivalMeta; color: "#f2c2cc"; font.pixelSize: 11; elide: Text.ElideRight
                                        Layout.fillWidth: true; Layout.minimumWidth: 0
                                    }
                                }
                                Label {
                                    objectName: "enemyMinimalistStats"
                                    visible: backend.minimalistMode
                                    color: "#ff9e9e"; font.pixelSize: 12; font.bold: true
                                    horizontalAlignment: Text.AlignRight
                                    text: (backend.showTotalDamage ? meter.minimalistDamageText(enemyBar.damage) + " · " : "") + meter.dpsText(enemyBar.dps) + " DPS" + (enemyBar.criticalKnown ? " · " + enemyBar.criticalRate.toFixed(0) + "%C" : "") + (enemyBar.maxHit > 0 ? " · Max " + meter.minimalistDamageText(enemyBar.maxHit) : "")
                                }
                                ColumnLayout {
                                    visible: !backend.minimalistMode
                                    Layout.preferredWidth: backend.barHeight < 38 ? Math.max(80, enemyBar.width / 2 - 24) : implicitWidth
                                    spacing: 0
                                    Label {
                                        text: meter.dpsText(enemyBar.dps) + " DPS"; color: "#ffdbe2"; font.bold: true; Layout.alignment: Qt.AlignRight
                                    }
                                    Label {
                                        visible: (backend.showTotalDamage || backend.showDetails) && backend.barHeight >= 38
                                        text: (backend.showTotalDamage ? meter.number(enemyBar.damage) + " dmg" : "") + (backend.showTotalDamage && backend.showDetails ? " · " : "") + (backend.showDetails ? enemyBar.share.toFixed(1) + "%" : "")
                                        color: "#f2c2cc"; font.pixelSize: 11; Layout.alignment: Qt.AlignRight
                                    }
                                }
                            }
                        }
                    }
                }
                RowLayout {
                    objectName: "meterFooter"
                    Layout.fillWidth: true; spacing: 8
                    Label {
                        objectName: "displayLabel"
                        text: backend.displayLabel; color: "#8e9eb5"; font.pixelSize: 11; elide: Text.ElideRight
                        Layout.fillWidth: true; Layout.minimumWidth: 0
                        ToolTip.visible: displayHover.hovered; ToolTip.text: backend.displayStatus
                        HoverHandler {id: displayHover}
                    }
                    Label {
                        objectName: "appVersionLabel"
                        text: "v" + backend.appVersion; color: "#8e9eb5"; font.pixelSize: 10
                    }
                    Label {
                        objectName: "updateAvailableLink"
                        visible: updates.available
                        text: "Update available"; color: "#9ad9ec"; font.pixelSize: 10; font.underline: true
                        Accessible.role: Accessible.Link; Accessible.name: "Update available: " + updates.version
                        TapHandler {onTapped: Qt.openUrlExternally(updates.url)}
                        HoverHandler {id: updateHover; cursorShape: Qt.PointingHandCursor}
                        ToolTip.visible: updateHover.hovered; ToolTip.text: "v" + updates.version + " · Open GitHub releases"
                    }
                    Rectangle {
                        objectName: "captureIndicator"
                        Layout.preferredWidth: 9; Layout.preferredHeight: 9; radius: 4.5
                        color: backend.captureActive ? "#54cf82" : "#ee6464"
                        Accessible.name: backend.captureActive ? "Capture active" : "Capture inactive"
                        HoverHandler {id: captureHover}
                        ToolTip.visible: captureHover.hovered; ToolTip.text: (backend.captureActive ? "Capture active" : "Capture inactive") + "\n" + backend.status
                    }
                }
            }
        }
    }
    onClosing: backend.quit()
}
