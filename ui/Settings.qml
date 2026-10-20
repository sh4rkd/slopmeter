import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
Window {
    id: settings
    objectName: "settingsWindow"
    visible: false
    width: 440; height: 740; minimumWidth: 360; minimumHeight: 400
    color: "#19202d"; flags: Qt.Window
    title: "SlopMeter · Settings"
    ScrollView {
        anchors.fill: parent; anchors.margins: 20
        clip: true; contentWidth: availableWidth
        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
        ColumnLayout {
            width: parent.width; spacing: 6
            Label {text: "Meter settings"; color: "#f2f5fa"; font.pixelSize: 22; font.bold: true}
            Label {text: "Changes apply immediately and are saved automatically."; color: "#9eadc3"; Layout.fillWidth: true; wrapMode: Text.WordWrap}
            Label {text: "Character"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            RowLayout {
                Layout.fillWidth: true
                TextField {id: characterNameField; objectName: "characterNameField"; Layout.fillWidth: true; placeholderText: "Character name"; text: backend.characterName; maximumLength: 72; onAccepted: backend.characterName = text}
                Button {text: "Save"; onClicked: backend.characterName = characterNameField.text}
            }
            Label {text: "Remembers the last detected name. Captured character identity overrides it on relog; name-based matches are rechecked after a zone change."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Layout"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Monitor / Pantalla de visualización"; color: "#c0cbdc"}
            ComboBox {
                id: screenSelector
                objectName: "screenSelector"
                Layout.fillWidth: true
                model: {
                    const list = [{ name: "__follow__", label: "Automático · Seguir pantalla de Aion 2" }]
                    const screens = backend.availableScreens || []
                    for (let i = 0; i < screens.length; ++i) {
                        list.push(screens[i])
                    }
                    return list
                }
                textRole: "label"
                valueRole: "name"
                currentIndex: {
                    if (backend.followGameDisplay) return 0
                    const cur = backend.selectedScreenName
                    const count = screenSelector.model.length
                    for (let i = 1; i < count; ++i) {
                        if (screenSelector.model[i].name === cur) return i
                    }
                    return 0
                }
                onActivated: {
                    if (currentValue === "__follow__") {
                        backend.followGameDisplay = true
                    } else {
                        backend.selectedScreenName = currentValue
                    }
                }
            }
            Label {
                text: "Elige en qué monitor (pantalla 1, pantalla 2, etc.) se mostrará SlopMeter o mantén la detección automática para que siga la ventana de Aion 2."
                color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap
            }
            CheckBox {objectName: "testModeToggle"; palette.windowText: "#e6edf6"; text: "Test mode"; checked: backend.testMode; onToggled: backend.testMode = checked}
            CheckBox {objectName: "minimalistModeToggle"; palette.windowText: "#e6edf6"; text: "Minimalist mode (text-only, no bars)"; checked: backend.minimalistMode; onToggled: backend.minimalistMode = checked}
            CheckBox {objectName: "minimalistCompactDamageToggle"; Layout.leftMargin: 20; visible: backend.minimalistMode; palette.windowText: "#e6edf6"; text: "Compact numbers — damage, CP/GS, HP (102.3K / 6.7M vs 102,301 / 6,741,514)"; checked: backend.minimalistCompactDamage; onToggled: backend.minimalistCompactDamage = checked}
            CheckBox {objectName: "borderToggle"; palette.windowText: "#e6edf6"; text: "Show meter border"; checked: backend.showBorder; onToggled: backend.showBorder = checked}
            Label {text: "Bar growth"; color: "#c0cbdc"}
            ComboBox {objectName: "barGrowthSelector"; Layout.fillWidth: true; model: ["Down · keep top edge fixed", "Up · keep bottom edge fixed"]; currentIndex: backend.growUp ? 1 : 0; onActivated: backend.growUp = currentIndex === 1}
            Label {text: "Player bar style"; color: "#c0cbdc"; enabled: !backend.minimalistMode}
            ComboBox {objectName: "barStyleSelector"; Layout.fillWidth: true; enabled: !backend.minimalistMode; model: ["Classic (default)", "Soft gradient", "Inset shadow", "Glossy"]; currentIndex: backend.barStyle; onActivated: backend.barStyle = currentIndex}
            Label {text: "Width · " + backend.barWidth + " px"; color: "#c0cbdc"}
            Slider {objectName: "barWidthSlider"; Layout.fillWidth: true; from: 240; to: 720; stepSize: 8; value: backend.barWidth; onMoved: backend.barWidth = Math.round(value)}
            Label {text: "Height · " + backend.barHeight + " px"; color: "#c0cbdc"; enabled: !backend.minimalistMode}
            Slider {objectName: "barHeightSlider"; Layout.fillWidth: true; enabled: !backend.minimalistMode; from: 28; to: 80; stepSize: 2; value: backend.barHeight; onMoved: backend.barHeight = Math.round(value)}
            Label {text: "Vertical spacing · " + backend.barSpacing + " px"; color: "#c0cbdc"}
            Slider {objectName: "barSpacingSlider"; Layout.fillWidth: true; from: 0; to: 20; stepSize: 1; value: backend.barSpacing; onMoved: backend.barSpacing = Math.round(value)}
            Label {text: "Radar de Cubos (Hidden Cubes)"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            CheckBox {objectName: "radarEnabledToggle"; palette.windowText: "#e6edf6"; text: "Activar Radar de Cubos Ocultos"; checked: backend.radarEnabled; onToggled: backend.radarEnabled = checked}
            CheckBox {objectName: "radarSimpleModeToggle"; Layout.leftMargin: 20; visible: backend.radarEnabled; palette.windowText: "#54cf82"; text: "Modo sencillo (Solo texto/alerta: hay un cubo a X metros, enfrente/atrás/izq/der/arriba/abajo)"; checked: backend.radarSimpleMode; onToggled: backend.radarSimpleMode = checked}
            CheckBox {objectName: "radarChestsToggle"; Layout.leftMargin: 20; visible: backend.radarEnabled; palette.windowText: "#f2c15a"; text: "Rastrear Hidden Cubes"; checked: backend.radarChests; onToggled: backend.radarChests = checked}
            CheckBox {objectName: "radarOdyleToggle"; visible: false; checked: false; onToggled: backend.radarOdyle = checked}
            CheckBox {objectName: "radarGemsToggle"; visible: false; checked: false; onToggled: backend.radarGems = checked}
            CheckBox {objectName: "radarOresToggle"; visible: false; checked: false; onToggled: backend.radarOres = checked}
            CheckBox {objectName: "radarHerbsToggle"; visible: false; checked: false; onToggled: backend.radarHerbs = checked}
            CheckBox {objectName: "radarWoodToggle"; visible: false; checked: false; onToggled: backend.radarWood = checked}
            CheckBox {objectName: "radarOtherToggle"; visible: false; checked: false; onToggled: backend.radarOther = checked}
            Label {text: "PvP Duel / Combate entre Jugadores"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            CheckBox {objectName: "pvpEnabledToggle"; palette.windowText: "#e6edf6"; text: "Activar modo PvP (barra de rivales, daño recibido vs infligido y duelo)"; checked: backend.pvpEnabled; onToggled: backend.pvpEnabled = checked}
            Label {text: "Alertas y Horarios de Eventos Aion 2"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            CheckBox {objectName: "eventAlertsEnabledToggle"; palette.windowText: "#e6edf6"; text: "Activar sistema de alertas de eventos periódicos"; checked: backend.eventAlertsEnabled; onToggled: backend.eventAlertsEnabled = checked}
            CheckBox {objectName: "eventSoundToggle"; Layout.leftMargin: 20; visible: backend.eventAlertsEnabled; palette.windowText: "#54cf82"; text: "Reproducir sonido de alerta en eventos próximos"; checked: backend.eventSoundEnabled; onToggled: backend.eventSoundEnabled = checked}
            RowLayout {
                Layout.leftMargin: 20; visible: backend.eventAlertsEnabled; Layout.fillWidth: true; spacing: 8
                Label {text: "Minutos de anticipación para avisar:"; color: "#c0cbdc"}
                SpinBox {
                    id: leadSpinBox
                    objectName: "eventLeadMinutesSpinBox"
                    from: 1; to: 30; stepSize: 1
                    value: backend.eventAlertLeadMinutes
                    onValueModified: backend.eventAlertLeadMinutes = value
                }
                Button {
                    text: "Probar Sonido"; onClicked: backend.playAlertSound("event")
                }
            }
            Label {text: "Eventos a rastrear individualmente:"; visible: backend.eventAlertsEnabled; color: "#9eadc3"; Layout.leftMargin: 20; font.pixelSize: 11}
            Repeater {
                model: backend.eventSchedule
                delegate: CheckBox {
                    required property var modelData
                    Layout.leftMargin: 24
                    visible: backend.eventAlertsEnabled
                    palette.windowText: "#e6edf6"
                    text: (modelData.icon || "") + " " + modelData.name + " (" + modelData.startTime + " · cada " + modelData.windowDuration + ")"
                    checked: modelData.tracked
                    onToggled: backend.setEventTracked(modelData.id, checked)
                }
            }
            Label {text: "Player bar information"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Gear score / combat power"; color: "#c0cbdc"}
            ComboBox {
                objectName: "scoreDisplaySelector"; Layout.fillWidth: true
                model: ["Hidden", "Gear score and combat power", "Combat power only", "Gear score only"]
                currentIndex: backend.scoreDisplay; onActivated: backend.scoreDisplay = currentIndex
            }
            CheckBox {objectName: "totalDamageToggle"; palette.windowText: "#e6edf6"; text: "Show total damage"; checked: backend.showTotalDamage; onToggled: backend.showTotalDamage = checked}
            CheckBox {palette.windowText: "#e6edf6"; text: "Show damage contribution (%)"; checked: backend.showDetails; onToggled: backend.showDetails = checked}
            CheckBox {objectName: "roundedDpsToggle"; palette.windowText: "#e6edf6"; text: "Rounded DPS (12.3K / 1.2M)"; checked: backend.roundedDps; onToggled: backend.roundedDps = checked}
            Label {text: "Opacity"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Background opacity · " + backend.backgroundOpacity + "%"; color: "#c0cbdc"}
            Slider {objectName: "backgroundOpacitySlider"; Layout.fillWidth: true; from: 0; to: 100; stepSize: 1; value: backend.backgroundOpacity; onMoved: backend.backgroundOpacity = Math.round(value)}
            Label {text: "Affects the blue background only."; color: "#9eadc3"; font.pixelSize: 11}
            Label {text: "Overall meter opacity · " + backend.overallOpacity + "%"; color: "#c0cbdc"}
            Slider {objectName: "overallOpacitySlider"; Layout.fillWidth: true; from: 0; to: 100; stepSize: 1; value: backend.overallOpacity; onMoved: backend.overallOpacity = Math.round(value)}
            Label {text: "Fades the background, bars and text together. Settings stays visible; reopen it from the tray if the meter is invisible."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Network capture"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            ComboBox {
                id: interfaceSelector; objectName: "captureInterfaceSelector"
                Layout.fillWidth: true; enabled: !backend.captureReplay
                model: backend.captureInterfaces; textRole: "label"; valueRole: "name"
                currentIndex: {
                    const rows = backend.captureInterfaces
                    const selected = backend.captureInterface
                    for (let i = 0; i < rows.length; ++i)
                        if (rows[i].name === selected) return i
                    return -1
                }
                onActivated: backend.captureInterface = currentValue
            }
            RowLayout {
                Button {text: "Refresh interfaces"; onClicked: backend.refreshInterfaces()}
                Button {text: "Apply and restart capture"; enabled: !backend.captureReplay; onClicked: backend.applyCaptureInterface()}
            }
            Label {text: backend.captureReplay ? "Interface selection is disabled during replay." : "Automatic uses the capture helper’s default interface. Choose your Ethernet, Wi-Fi, VPN, or all interfaces if needed. Applying ends the current fight and may request capture permission again."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: backend.status; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: backend.interfaceStatus; visible: text.length > 0; color: "#c0cbdc"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Boss health: " + backend.bossHealthStatus; visible: backend.bossHealthStatus.length > 0; color: "#c0cbdc"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Data refresh"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Data polling interval · " + backend.pollInterval + " ms"; color: "#c0cbdc"}
            Slider {objectName: "pollIntervalSlider"; Layout.fillWidth: true; from: 50; to: 1000; stepSize: 50; snapMode: Slider.SnapAlways; value: backend.pollInterval; onMoved: backend.pollInterval = Math.round(value)}
            Label {text: "Packets are captured continuously. This controls how often the meter refreshes. Default: 200 ms."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {objectName: "cpuWarning"; visible: backend.pollInterval < 200; text: "Warning: intervals below 200 ms can cause high CPU usage."; color: "#f1c778"; Layout.fillWidth: true; font.pixelSize: 12; wrapMode: Text.WordWrap}
            Label {text: "SlopMeter " + backend.appVersion; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            RowLayout {
                Layout.fillWidth: true; Layout.topMargin: 10
                Button {text: "Restore defaults"; onClicked: backend.resetAppearance()}
                Item {Layout.fillWidth: true}
                Button {text: "Close"; onClicked: settings.close()}
            }
        }
    }
}
