#include <QApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQmlProperty>
#include <QQuickWindow>
#include <QProcess>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QTimer>
#include <QSettings>
#include <QScreen>
#include <QAbstractListModel>
#include <QElapsedTimer>
#include <QSystemTrayIcon>
#include <QMenu>
#include <QPainter>
#include <LayerShellQt/Window>
static bool g_noLayerShell=false;
#include <wayland-client.h>
#include <cstdio>
#include <cstring>
#include <QFile>
#include <QFileInfo>
#include <QDir>
#include <QTemporaryFile>
#include <QTemporaryDir>
#include <memory>
#include <cmath>
#include <QDBusConnection>
#include <QDBusInterface>
#include <QDBusPendingCallWatcher>
#include <QDBusPendingReply>
#include <QQuickItem>
#include <QQuickItemGrabResult>
#include <QMessageBox>
#include <QRegion>
#include <QSet>
#include <QEventLoop>
#include <QDateTime>
#include <QStandardPaths>
#include <QUrl>
#include <QNetworkAccessManager>
#include <QNetworkRequest>
#include <QNetworkReply>
#include <QVBoxLayout>
#include <QPushButton>
#include <QLabel>
#include <functional>
#include "desktop.h"
#include "releasecheck.h"
#include "desktopregistration.h"

class ActorModel : public QAbstractListModel {
public:
    QList<QVariantMap> rows;
    enum { Name=Qt::UserRole+1, Damage, DamageTaken, DPS, CriticalRate, CriticalKnown, MaxHit, Enemy, Share, Fill, Class, ActorId, GearScore, CombatPower };
    using QAbstractListModel::QAbstractListModel;
    int rowCount(const QModelIndex &parent={}) const override { return parent.isValid()?0:rows.size(); }
    QHash<int,QByteArray> roleNames() const override {static const QHash<int,QByteArray> roles{{Name,"actorName"},{Damage,"damage"},{DamageTaken,"damageTaken"},{DPS,"dps"},{CriticalRate,"criticalRate"},{CriticalKnown,"criticalKnown"},{MaxHit,"maxHit"},{Enemy,"enemy"},{Share,"share"},{Fill,"fill"},{Class,"actorClass"},{ActorId,"actorId"},{GearScore,"gearScore"},{CombatPower,"combatPower"}};return roles;}
    QVariant data(const QModelIndex &i,int role) const override {
        if (!i.isValid()||i.row()>=rows.size()) return {};
        if(role==GearScore || role==CombatPower || role==DamageTaken || role==CriticalRate || role==MaxHit) return rows[i.row()].value(QString::fromUtf8(roleNames().value(role)), 0);
        if(role==CriticalKnown || role==Enemy) return rows[i.row()].value(QString::fromUtf8(roleNames().value(role)), false);
        return rows[i.row()].value(QString::fromUtf8(roleNames().value(role)) == "actorName" ? "name" : role==Class ? "class" : role==ActorId ? "id" : QString::fromUtf8(roleNames().value(role)));
    }
    void update(const QJsonArray &array) {
        // Remove departing rows before insertion to avoid transient panel growth
        // when switching between real and simulated parties. Keep surviving delegates.
        QSet<qulonglong> ids;for(const auto &value:array) ids.insert(value.toObject().value("id").toVariant().toULongLong());
        for(int i=rows.size()-1;i>=0;--i) if(!ids.contains(rows[i].value("id").toULongLong())) {beginRemoveRows({},i,i);rows.removeAt(i);endRemoveRows();}
        for(int i=0;i<array.size();++i) {
            auto row=array[i].toObject().toVariantMap(); int existing=-1;
            for(int j=i;j<rows.size();++j) if(rows[j].value("id")==row.value("id")) { existing=j; break; }
            if(existing<0) { beginInsertRows({},i,i);rows.insert(i,row);endInsertRows(); }
            else {
                if(existing!=i) { beginMoveRows({},existing,existing,{},i);rows.move(existing,i);endMoveRows(); }
                if(rows[i]!=row) { rows[i]=row;emit dataChanged(index(i),index(i)); }
            }
        }
        if(rows.size()>array.size()) { beginRemoveRows({},array.size(),rows.size()-1);while(rows.size()>array.size()) rows.removeLast();endRemoveRows(); }
    }
};
class SkillModel : public ActorModel {
public:
    using ActorModel::ActorModel;
    QHash<int,QByteArray> roleNames() const override {
        static const QHash<int,QByteArray> roles=[] {
            QHash<int,QByteArray> roles;int key=Qt::UserRole+101;
            for(const auto &name:QList<QByteArray>{"skillId","skillName","skillIcon","damage","share","fightShare","hits","uses","usesKnown","critical","criticalRate","criticalKnown","minDamage","maxDamage","average","hitsPerSecond"}) roles.insert(key++,name);
            return roles;
        }();return roles;
    }
    QVariant data(const QModelIndex &i,int role) const override {
        if(!i.isValid()||i.row()>=rows.size()) return {};
        auto key=QString::fromUtf8(roleNames().value(role));
        if(key=="skillId") key="id";else if(key=="skillName") key="name";else if(key=="skillIcon") key="icon";else if(key=="minDamage") key="min";else if(key=="maxDamage") key="max";
        return rows[i.row()].value(key);
    }
};
class Bridge : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList fightChoices READ fightChoices NOTIFY historyChanged)
    Q_PROPERTY(int selectedFightIndex READ selectedFightIndex NOTIFY historyChanged)
    Q_PROPERTY(QVariantList displayPlayers READ displayPlayers NOTIFY detailChanged)
    Q_PROPERTY(QVariantList displayEnemies READ displayEnemies NOTIFY detailChanged)
    Q_PROPERTY(QVariantList detailChoices READ detailChoices NOTIFY detailChanged)
    Q_PROPERTY(bool pvpEnabled READ pvpEnabled WRITE setPvpEnabled NOTIFY pvpSettingChanged)
    Q_PROPERTY(bool isPvP READ isPvP NOTIFY pvpChanged)
    Q_PROPERTY(QVariantMap pvpSummary READ pvpSummary NOTIFY pvpChanged)
    Q_PROPERTY(bool radarSimpleMode READ radarSimpleMode WRITE setRadarSimpleMode NOTIFY radarChanged)
    Q_PROPERTY(bool radarEnabled READ radarEnabled WRITE setRadarEnabled NOTIFY radarChanged)
    Q_PROPERTY(bool radarChests READ radarChests WRITE setRadarChests NOTIFY radarChanged)
    Q_PROPERTY(bool radarOdyle READ radarOdyle WRITE setRadarOdyle NOTIFY radarChanged)
    Q_PROPERTY(bool radarGems READ radarGems WRITE setRadarGems NOTIFY radarChanged)
    Q_PROPERTY(bool radarOres READ radarOres WRITE setRadarOres NOTIFY radarChanged)
    Q_PROPERTY(bool radarHerbs READ radarHerbs WRITE setRadarHerbs NOTIFY radarChanged)
    Q_PROPERTY(bool radarWood READ radarWood WRITE setRadarWood NOTIFY radarChanged)
    Q_PROPERTY(bool radarResources READ radarResources WRITE setRadarResources NOTIFY radarChanged)
    Q_PROPERTY(bool radarOther READ radarOther WRITE setRadarOther NOTIFY radarChanged)
    Q_PROPERTY(QString radarItemFilter READ radarItemFilter WRITE setRadarItemFilter NOTIFY radarChanged)
    Q_PROPERTY(QVariantList displayRadar READ displayRadar NOTIFY radarChanged)
    Q_PROPERTY(int radarSelectedId READ radarSelectedId WRITE setRadarSelectedId NOTIFY radarSelectionChanged)
    Q_PROPERTY(double radarRangeMeters READ radarRangeMeters WRITE setRadarRangeMeters NOTIFY radarRangeChanged)
    Q_PROPERTY(QString radarSubmenu READ radarSubmenu WRITE setRadarSubmenu NOTIFY radarSubmenuChanged)
    Q_PROPERTY(bool eventAlertsEnabled READ eventAlertsEnabled WRITE setEventAlertsEnabled NOTIFY eventAlertsChanged)
    Q_PROPERTY(bool eventSoundEnabled READ eventSoundEnabled WRITE setEventSoundEnabled NOTIFY eventAlertsChanged)
    Q_PROPERTY(int eventAlertLeadMinutes READ eventAlertLeadMinutes WRITE setEventAlertLeadMinutes NOTIFY eventAlertsChanged)
    Q_PROPERTY(QVariantList eventSchedule READ eventSchedule NOTIFY eventAlertsChanged)
    Q_PROPERTY(QVariantMap activeEventAlert READ activeEventAlert NOTIFY eventAlertsChanged)
    Q_PROPERTY(QVariantList marketServers READ marketServers NOTIFY marketChanged)
    Q_PROPERTY(int selectedMarketServer READ selectedMarketServer WRITE setSelectedMarketServer NOTIFY marketChanged)
    Q_PROPERTY(QString selectedMarketServerName READ selectedMarketServerName NOTIFY marketChanged)
    Q_PROPERTY(bool marketLoading READ marketLoading NOTIFY marketChanged)
    Q_PROPERTY(QString marketStatus READ marketStatus NOTIFY marketChanged)
    Q_PROPERTY(double marketTaxRate READ marketTaxRate WRITE setMarketTaxRate NOTIFY marketChanged)
    Q_PROPERTY(QVariantList craftRecipes READ craftRecipes NOTIFY marketChanged)
    Q_PROPERTY(QVariantMap detailPlayer READ detailPlayer NOTIFY detailChanged)
    Q_PROPERTY(QVariantMap detailFight READ detailFight NOTIFY detailChanged)
    Q_PROPERTY(bool testMode READ testMode WRITE setTestMode NOTIFY testModeChanged)
    Q_PROPERTY(QString appVersion READ appVersion CONSTANT)
    Q_PROPERTY(int barStyle READ barStyle WRITE setBarStyle NOTIFY appearanceChanged)
    Q_PROPERTY(bool minimalistMode READ minimalistMode WRITE setMinimalistMode NOTIFY appearanceChanged)
    Q_PROPERTY(bool minimalistCompactDamage READ minimalistCompactDamage WRITE setMinimalistCompactDamage NOTIFY appearanceChanged)
    Q_PROPERTY(bool growUp READ growUp WRITE setGrowUp NOTIFY appearanceChanged)
    Q_PROPERTY(bool showBorder READ showBorder WRITE setShowBorder NOTIFY appearanceChanged)
    Q_PROPERTY(QString captureInterface READ captureInterface WRITE setCaptureInterface NOTIFY interfacesChanged)
    Q_PROPERTY(QVariantList captureInterfaces READ captureInterfaces NOTIFY interfacesChanged)
    Q_PROPERTY(bool captureReplay READ captureReplay NOTIFY interfacesChanged)
    Q_PROPERTY(QString interfaceStatus READ interfaceStatus NOTIFY interfacesChanged)
    Q_PROPERTY(int pollInterval READ pollInterval WRITE setPollInterval NOTIFY pollingChanged)
    Q_PROPERTY(int barWidth READ barWidth WRITE setBarWidth NOTIFY appearanceChanged)
    Q_PROPERTY(int barHeight READ barHeight WRITE setBarHeight NOTIFY appearanceChanged)
    Q_PROPERTY(int barSpacing READ barSpacing WRITE setBarSpacing NOTIFY appearanceChanged)
    Q_PROPERTY(int backgroundOpacity READ backgroundOpacity WRITE setBackgroundOpacity NOTIFY appearanceChanged)
    Q_PROPERTY(int overallOpacity READ overallOpacity WRITE setOverallOpacity NOTIFY appearanceChanged)
    Q_PROPERTY(bool showDetails READ showDetails WRITE setShowDetails NOTIFY appearanceChanged)
    Q_PROPERTY(int scoreDisplay READ scoreDisplay WRITE setScoreDisplay NOTIFY appearanceChanged)
    Q_PROPERTY(bool showTotalDamage READ showTotalDamage WRITE setShowTotalDamage NOTIFY appearanceChanged)
    Q_PROPERTY(bool roundedDps READ roundedDps WRITE setRoundedDps NOTIFY appearanceChanged)
    Q_PROPERTY(bool captureActive READ captureActive NOTIFY captureChanged)
    Q_PROPERTY(QString displayLabel READ displayLabel NOTIFY displayChanged)
    Q_PROPERTY(QVariantList availableScreens READ availableScreens NOTIFY displayChanged)
    Q_PROPERTY(QString selectedScreenName READ selectedScreenName WRITE setSelectedScreenName NOTIFY displayChanged)
    Q_PROPERTY(bool followGameDisplay READ followGameDisplay WRITE setFollowGameDisplay NOTIFY displayChanged)
    Q_PROPERTY(QString character READ character NOTIFY characterChanged)
    Q_PROPERTY(QString characterName READ characterName WRITE setCharacterName NOTIFY characterNameChanged)
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(double duration READ duration NOTIFY durationChanged)
    Q_PROPERTY(QString encounter READ encounter NOTIFY encounterChanged)
    Q_PROPERTY(QString sceneName READ sceneName NOTIFY encounterChanged)
    Q_PROPERTY(QVariantMap boss READ boss NOTIFY bossChanged)
    Q_PROPERTY(QString bossHealthStatus READ bossHealthStatus NOTIFY bossChanged)
    Q_PROPERTY(QString displayStatus READ displayStatus NOTIFY displayChanged)
    Q_CLASSINFO("D-Bus Interface", "org.aiondps.Screen")
public:
    bool previewEnabled=false;
    bool borderSetting=QSettings().value("appearance/showBorder",true).toBool();
    int pollSetting=QSettings().value("capture/pollIntervalMs",200).toInt();
    QElapsedTimer previewClock;
    QVariantList previewParty;
    QList<double> previewDamage;
    double previewLast=0;
 double previewBaseSeconds=60;
    quint32 actorBeforePreview=0;
    static int normalizePoll(int value) {return ((qBound(50,value,1000)+25)/50)*50;}
    bool testMode() const {return previewEnabled;}
    int styleSetting=QSettings().value("appearance/barStyle",0).toInt();
    bool minimalistSetting=QSettings().value("appearance/minimalistMode",false).toBool();
    bool minimalistCompactDamageSetting=QSettings().value("appearance/minimalistCompactDamage",true).toBool();
    bool growthSetting=QSettings().value("appearance/growUp",false).toBool();
    bool radarEnabledSetting=QSettings().value("radar/enabled",false).toBool();
    bool radarChestsSetting=QSettings().value("radar/chests",true).toBool();
    bool radarOdyleSetting=QSettings().value("radar/odyle",true).toBool();
    bool radarGemsSetting=QSettings().value("radar/gems",true).toBool();
    bool radarOresSetting=QSettings().value("radar/ores",true).toBool();
    bool radarHerbsSetting=QSettings().value("radar/herbs",true).toBool();
    bool radarWoodSetting=QSettings().value("radar/wood",true).toBool();
    bool radarOtherSetting=QSettings().value("radar/other",false).toBool();
    QString radarItemFilterSetting=QSettings().value("radar/itemFilter",QString()).toString();
    bool radarSimpleModeSetting=QSettings().value("radar/simpleMode",false).toBool();
    bool radarSimpleMode() const {return radarSimpleModeSetting;}
    void setRadarSimpleMode(bool value) {if(radarSimpleModeSetting==value) return;radarSimpleModeSetting=value;QSettings().setValue("radar/simpleMode",value);emit radarChanged();}
    bool radarEnabled() const {return radarEnabledSetting;}
    void setRadarEnabled(bool value) {if(radarEnabledSetting==value) return;radarEnabledSetting=value;QSettings().setValue("radar/enabled",value);emit radarChanged();}
    bool radarChests() const {return radarChestsSetting;}
    void setRadarChests(bool value) {if(radarChestsSetting==value) return;radarChestsSetting=value;QSettings().setValue("radar/chests",value);emit radarChanged();}
    bool radarOdyle() const {return radarOdyleSetting;}
    void setRadarOdyle(bool value) {if(radarOdyleSetting==value) return;radarOdyleSetting=value;QSettings().setValue("radar/odyle",value);emit radarChanged();}
    bool radarGems() const {return radarGemsSetting;}
    void setRadarGems(bool value) {if(radarGemsSetting==value) return;radarGemsSetting=value;QSettings().setValue("radar/gems",value);emit radarChanged();}
    bool radarOres() const {return radarOresSetting;}
    void setRadarOres(bool value) {if(radarOresSetting==value) return;radarOresSetting=value;QSettings().setValue("radar/ores",value);emit radarChanged();}
    bool radarHerbs() const {return radarHerbsSetting;}
    void setRadarHerbs(bool value) {if(radarHerbsSetting==value) return;radarHerbsSetting=value;QSettings().setValue("radar/herbs",value);emit radarChanged();}
    bool radarWood() const {return radarWoodSetting;}
    void setRadarWood(bool value) {if(radarWoodSetting==value) return;radarWoodSetting=value;QSettings().setValue("radar/wood",value);emit radarChanged();}
    bool radarResources() const {return radarGemsSetting||radarOresSetting||radarHerbsSetting||radarWoodSetting;}
    void setRadarResources(bool value) {setRadarGems(value);setRadarOres(value);setRadarHerbs(value);setRadarWood(value);}
    bool radarOther() const {return radarOtherSetting;}
    void setRadarOther(bool value) {if(radarOtherSetting==value) return;radarOtherSetting=value;QSettings().setValue("radar/other",value);emit radarChanged();}
    QString radarItemFilter() const {return radarItemFilterSetting;}
    void setRadarItemFilter(const QString &value) {if(radarItemFilterSetting==value) return;radarItemFilterSetting=value;QSettings().setValue("radar/itemFilter",value);emit radarChanged();}
    int radarSelectedIdSetting=0;
    double radarRangeMetersSetting=110.0;
    QString radarSubmenuSetting="";
    int radarSelectedId() const {return radarSelectedIdSetting;}
    void setRadarSelectedId(int val) {if(radarSelectedIdSetting==val) return;radarSelectedIdSetting=val;emit radarSelectionChanged();}
    double radarRangeMeters() const {return radarRangeMetersSetting;}
    void setRadarRangeMeters(double val) {if(qFuzzyCompare(radarRangeMetersSetting,val)) return;radarRangeMetersSetting=val;emit radarRangeChanged();}
    QString radarSubmenu() const {return radarSubmenuSetting;}
    void setRadarSubmenu(const QString &val) {if(radarSubmenuSetting==val) return;radarSubmenuSetting=val;emit radarSubmenuChanged();}
    Q_INVOKABLE QStringList radarSubItemsFor(const QString &catKey) const {
        if(catKey=="gem") return {"Sapphire Gemstone", "Ruby Gemstone", "Diamond Gemstone"};
        if(catKey=="ore") return {"Orichalcum Ore"};
        if(catKey=="herb") return {"Azpha", "Calendula", "Pujery", "Conide", "Ohkra", "Morfa", "Raydam", "Aria", "Targena", "Mela", "Coriolus", "Inina", "Cypri", "Kukuru"};
        if(catKey=="wood") return {"Asvata Log", "Yggdrasil Log"};
        if(catKey=="odyle") return {"Odyle"};
        if(catKey=="chest") return {"Hidden Cube"};
        return {};
    }
    Q_INVOKABLE QVariantMap selectedRadarTarget() const {
        const auto rows=displayRadar();
        if(rows.isEmpty()) return {};
        if(radarSelectedIdSetting!=0) {
            for(const auto &item:rows) {
                const auto map=item.toMap();
                if(map.value("id").toInt()==radarSelectedIdSetting) return map;
            }
        }
        return rows.first().toMap();
    }
    Q_INVOKABLE QString radarCategoryColor(const QString &cat) const {
        if(cat=="chest") return "#f2c15a";
        if(cat=="odyle") return "#6be0ff";
        if(cat=="gem") return "#e082ff";
        if(cat=="ore") return "#ff9f5a";
        if(cat=="herb" || cat=="resource") return "#7be35a";
        if(cat=="wood") return "#c89b6e";
        return "#c0cbdc";
    }
    Q_INVOKABLE QString radarCategoryIcon(const QString &cat) const {
        if(cat=="chest") return "◈";
        if(cat=="odyle") return "✦";
        if(cat=="gem") return "◆";
        if(cat=="ore") return "⬡";
        if(cat=="herb" || cat=="resource") return "❖";
        if(cat=="wood") return "▤";
        return "○";
    }
    Q_INVOKABLE QString radarVertShort(const QString &vDir) const {
        if(vDir=="Arriba") return "▲";
        if(vDir=="Abajo") return "▼";
        return "";
    }
    Q_INVOKABLE QString radarVertBadge(const QString &vDir, double dz) const {
        int d=qRound(qAbs(dz));
        if(vDir=="Arriba") return QString("▲ ARRIBA (+%1m)").arg(d);
        if(vDir=="Abajo") return QString("▼ ABAJO (-%1m)").arg(d);
        return QString("• MISMO NIVEL (%1%2m)").arg(dz>=0?"+":"", QString::number(d));
    }
    Q_INVOKABLE QString radarHorizArrow(const QString &hDir) const {
        if(hDir=="Norte" || hDir=="Frente") return "↑ NORTE (FRENTE)";
        if(hDir=="Sur" || hDir=="Atrás") return "↓ SUR (ATRÁS)";
        if(hDir=="Este" || hDir=="Der") return "→ ESTE (DER)";
        if(hDir=="Oeste" || hDir=="Izq") return "← OESTE (IZQ)";
        if(hDir=="Noreste") return "↗ NORESTE";
        if(hDir=="Sureste") return "↘ SURESTE";
        if(hDir=="Noroeste") return "↖ NOROESTE";
        if(hDir=="Suroeste") return "↙ SUROESTE";
        return "● CENTRO";
    }
    bool minimalistMode() const {return minimalistSetting;}
    void setMinimalistMode(bool value) {if(minimalistSetting==value) return;minimalistSetting=value;QSettings().setValue("appearance/minimalistMode",value);emit appearanceChanged();}
    bool minimalistCompactDamage() const {return minimalistCompactDamageSetting;}
    void setMinimalistCompactDamage(bool value) {if(minimalistCompactDamageSetting==value) return;minimalistCompactDamageSetting=value;QSettings().setValue("appearance/minimalistCompactDamage",value);emit appearanceChanged();}
    bool growUp() const {return growthSetting;}
    void setGrowUp(bool value) {growthSetting=value;QSettings().setValue("appearance/growUp",value);savePosition();emit appearanceChanged();}
    int barStyle() const {return qBound(0,styleSetting,3);}
    void setBarStyle(int value) {styleSetting=qBound(0,value,3);QSettings().setValue("appearance/barStyle",styleSetting);emit appearanceChanged();}
    bool showBorder() const {return borderSetting;}
    int pollInterval() const {return normalizePoll(pollSetting);}
    void setShowBorder(bool value) {borderSetting=value;QSettings().setValue("appearance/showBorder",value);emit appearanceChanged();}
    Q_INVOKABLE void resetDamage() {
        if(previewEnabled) {
            for(auto &damage:previewDamage) damage=0;previewBaseSeconds=0;previewLast=0;previewClock.restart();refreshDisplay();return;
        }
        selectedSession=0;emit historyChanged();
        if(process.state()!=QProcess::NotRunning)
            process.write(QJsonDocument(QJsonObject{{"reset",true}}).toJson(QJsonDocument::Compact)+"\n");
        else {
            currentState["actors"]=QVariantList{};currentState["enemies"]=QVariantList{};
            currentState["pvp"]=false;currentState["pvpSummary"]=QVariantMap{};
            currentState["active"]=false;
            currentState["duration"]=0;currentState["cleared"]=true;
            currentState["encounter"]="Manually reset · Waiting for combat";
            refreshDisplay();
        }
    }
    void sendRefreshInterval() {
        if(process.state()!=QProcess::NotRunning) process.write(QJsonDocument(QJsonObject{{"intervalMs",pollInterval()}}).toJson(QJsonDocument::Compact)+"\n");
    }
    void setPollInterval(int value) {
        value=normalizePoll(value);if(value==pollInterval()) return;
        pollSetting=value;QSettings().setValue("capture/pollIntervalMs",value);
        publish.setInterval(value);sendRefreshInterval();emit pollingChanged();
    }
    void setTestMode(bool enabled) {
        if(previewEnabled==enabled) return;
        previewEnabled=enabled;
        if(enabled) {
            actorBeforePreview=selectedActor;selectedActor=0;previewBaseSeconds=60;previewClock.start();previewLast=0;previewDamage.clear();
            for(const auto &value:previewParty) previewDamage.append(value.toMap().value("baseDps").toDouble()*60);
        } else selectedActor=actorBeforePreview;
        if(enabled) publish.start();else publish.stop();
        refreshDisplay();emit testModeChanged();emit historyChanged();emit statusChanged();
    }
    QVariantMap previewState() {
        const double elapsed=previewClock.elapsed()/1000.0,seconds=qMax(1.0,previewBaseSeconds+elapsed),delta=qMax(0.0,elapsed-previewLast);
        double largest=0,totalDamage=0;
        for(int i=0;i<previewParty.size();++i) {
            previewDamage[i]+=delta*previewParty[i].toMap().value("baseDps").toDouble()*(1+0.08*std::sin(elapsed*0.5+i));
            largest=qMax(largest,previewDamage[i]);totalDamage+=previewDamage[i];
        }
        previewLast=elapsed;QVariantList rows;
        for(int i=0;i<previewParty.size();++i) {
            auto row=previewParty[i].toMap();const double damage=previewDamage[i];
            row["damage"]=qRound64(damage);row["dps"]=damage/seconds;row["fill"]=largest>0?damage/largest*100:0;row["share"]=totalDamage>0?damage/totalDamage*100:0;
            QVariantList skills;
            for(const auto &value:row.value("skills").toList()) {
                auto skill=value.toMap();const double skillDamage=damage*skill.value("weight").toDouble();
                const auto uses=qMax(1,qRound(seconds/3*skill.value("weight").toDouble()));const int hits=uses*2;
                skill["damage"]=qRound64(skillDamage);skill["share"]=damage>0?skillDamage/damage*100:0;skill["fightShare"]=totalDamage>0?skillDamage/totalDamage*100:0;
                skill["hits"]=hits;skill["uses"]=uses;skill["usesKnown"]=true;skill["critical"]=qRound(hits*0.3);skill["criticalRate"]=30.0;skill["criticalKnown"]=true;
                skill["min"]=qRound(skillDamage/hits*0.75);skill["max"]=qRound(skillDamage/hits*1.25);skill["average"]=skillDamage/hits;skill["hitsPerSecond"]=hits/seconds;
                skills.append(skill);
            }
            QVariantList timeline;
            // Synthetic casts for preview only, using the same timeline schema as capture.
            for(int cast=0;cast<qMin(5000,int(seconds/1.5));++cast) {
                if(skills.isEmpty()) break;
                const auto skill=skills[cast % skills.size()].toMap();
                timeline.append(QVariantMap{{"time",cast*1.5},{"skill",skill.value("id")},{"name",skill.value("name")},{"icon",skill.value("icon")}});
            }
            row["timeline"]=timeline;
            row["skills"]=skills;rows.append(row);
        }
        const qlonglong maximumHP=8000000;
        const qlonglong currentHP=qRound64(maximumHP*qMax(0.05,0.68-elapsed*0.002));
        const double fightDuration=previewBaseSeconds+elapsed;
        const double partyDps=totalDamage/seconds;
        const double eta=partyDps>0?currentHP/partyDps:0.0;
        const double totalEta=partyDps>0?fightDuration+eta:fightDuration;
        const double timeLeft=qMax(0.0,300.0-fightDuration);
        QVariantMap previewBoss{{"entity",900100},{"name","Fediv Wraith"},{"hp",currentHP},{"max",maximumHP},
            {"percent",100.0*currentHP/maximumHP},{"eta",eta},{"totalEta",totalEta},{"timeLeft",timeLeft},{"etaKnown",partyDps>0},{"known",true},{"maxKnown",true}};
        QVariantList previewRadar{
            QVariantMap{{"id",501},{"npc",1200039},{"name","Hidden Cube"},{"category","chest"},{"distance",18.4},{"zDelta",4.8},{"angle",12.0},{"relX",3.8},{"relY",18.0},{"horizDir","Frente"},{"vertDir","Arriba"},{"hasDir",true}},
            QVariantMap{{"id",502},{"npc",1080001},{"name","Odyle"},{"category","odyle"},{"distance",26.1},{"zDelta",-3.5},{"angle",82.0},{"relX",25.8},{"relY",3.6},{"horizDir","Der"},{"vertDir","Abajo"},{"hasDir",true}},
            QVariantMap{{"id",503},{"npc",1080501},{"name","Sapphire Gemstone"},{"category","gem"},{"distance",22.0},{"zDelta",3.2},{"angle",-35.0},{"relX",-12.6},{"relY",18.0},{"horizDir","Frente"},{"vertDir","Arriba"},{"hasDir",true}},
            QVariantMap{{"id",504},{"npc",1080101},{"name","Orichalcum Ore"},{"category","ore"},{"distance",34.0},{"zDelta",0.5},{"angle",-75.0},{"relX",-32.8},{"relY",8.8},{"horizDir","Izq"},{"vertDir","Nivel"},{"hasDir",true}},
            QVariantMap{{"id",505},{"npc",1090601},{"name","Azpha"},{"category","herb"},{"distance",45.0},{"zDelta",-5.0},{"angle",155.0},{"relX",19.0},{"relY",-40.8},{"horizDir","Atrás"},{"vertDir","Abajo"},{"hasDir",true}},
            QVariantMap{{"id",506},{"npc",1090201},{"name","Asvata Log"},{"category","wood"},{"distance",38.5},{"zDelta",1.0},{"angle",-130.0},{"relX",-29.5},{"relY",-24.7},{"horizDir","Izq"},{"vertDir","Nivel"},{"hasDir",true}}
        };
        return {{"character","Test party"},{"encounter","Test mode"},{"duration",fightDuration},{"active",true},{"session",0},{"actors",rows},{"boss",previewBoss},{"radar",previewRadar},{"scene",QVariantMap{{"id",600021},{"name","Fire Temple"},{"instance",1}}}};
    }
    int widthSetting=QSettings().value("appearance/barWidth",352).toInt();
    int heightSetting=QSettings().value("appearance/barHeight",44).toInt();
    int spacingSetting=QSettings().value("appearance/barSpacing",4).toInt();
    int backgroundSetting=QSettings().value("appearance/backgroundOpacity",75).toInt();
    int overallSetting=QSettings().value("appearance/overallOpacity",100).toInt();
    bool detailsSetting=QSettings().value("appearance/showDetails",true).toBool();
    int scoreSetting=QSettings().value("appearance/scoreDisplay",1).toInt();
    bool totalDamageSetting=QSettings().value("appearance/showTotalDamage",detailsSetting).toBool();
    bool roundedDpsSetting=QSettings().value("appearance/roundedDps",false).toBool();
    int barWidth() const {return qBound(240,widthSetting,720);}
    int barHeight() const {return qBound(28,heightSetting,80);}
    int barSpacing() const {return qBound(0,spacingSetting,20);}
    int backgroundOpacity() const {return qBound(0,backgroundSetting,100);}
    int overallOpacity() const {return qBound(0,overallSetting,100);}
    void setBackgroundOpacity(int value) {backgroundSetting=qBound(0,value,100);QSettings().setValue("appearance/backgroundOpacity",backgroundSetting);emit appearanceChanged();}
    void setOverallOpacity(int value) {overallSetting=qBound(0,value,100);QSettings().setValue("appearance/overallOpacity",overallSetting);emit appearanceChanged();}
    bool showDetails() const {return detailsSetting;}
    int scoreDisplay() const {return qBound(0,scoreSetting,3);}
    bool showTotalDamage() const {return totalDamageSetting;}
    bool roundedDps() const {return roundedDpsSetting;}
    void setScoreDisplay(int value) {scoreSetting=qBound(0,value,3);QSettings().setValue("appearance/scoreDisplay",scoreSetting);emit appearanceChanged();}
    void setShowTotalDamage(bool value) {totalDamageSetting=value;QSettings().setValue("appearance/showTotalDamage",value);emit appearanceChanged();}
    void setRoundedDps(bool value) {roundedDpsSetting=value;QSettings().setValue("appearance/roundedDps",value);emit appearanceChanged();}
    void setBarWidth(int value) {widthSetting=qBound(240,value,720);QSettings().setValue("appearance/barWidth",widthSetting);emit appearanceChanged();}
    void setBarHeight(int value) {heightSetting=qBound(28,value,80);QSettings().setValue("appearance/barHeight",heightSetting);emit appearanceChanged();}
    void setBarSpacing(int value) {spacingSetting=qBound(0,value,20);QSettings().setValue("appearance/barSpacing",spacingSetting);emit appearanceChanged();}
    void setShowDetails(bool value) {detailsSetting=value;QSettings().setValue("appearance/showDetails",value);emit appearanceChanged();}
    Q_INVOKABLE void resetAppearance() {setMinimalistMode(false);setMinimalistCompactDamage(true);setGrowUp(false);setScoreDisplay(1);setShowTotalDamage(true);setRoundedDps(false);setBarWidth(352);setBarHeight(44);setBarSpacing(4);setShowDetails(true);setBackgroundOpacity(75);setOverallOpacity(100);setShowBorder(true);setBarStyle(0);setPollInterval(200);setRadarEnabled(false);setRadarChests(true);setRadarOdyle(true);setRadarGems(true);setRadarOres(true);setRadarHerbs(true);setRadarWood(true);setRadarOther(false);setRadarItemFilter(QString());}
    ActorModel actors;
    ActorModel enemies;
    SkillModel skills;
    QProcess process;
    QByteArray buffer;
    QJsonObject pending;
    QTimer publish;
    QTimer x11RaiseTimer;
    QTimer eventTimer;
    QSet<QString> triggeredEventAlerts;
    QVariantMap currentActiveAlert;
    QDateTime currentActiveAlertUntil;
    bool eventAlertsEnabledSetting=QSettings().value("events/enabled",true).toBool();
    bool eventSoundEnabledSetting=QSettings().value("events/soundEnabled",true).toBool();
    int eventAlertLeadMinutesSetting=QSettings().value("events/leadMinutes",5).toInt();
    bool eventAlertsEnabled() const {return eventAlertsEnabledSetting;}
    void setEventAlertsEnabled(bool val) {if(eventAlertsEnabledSetting==val) return;eventAlertsEnabledSetting=val;QSettings().setValue("events/enabled",val);emit eventAlertsChanged();}
    bool eventSoundEnabled() const {return eventSoundEnabledSetting;}
    void setEventSoundEnabled(bool val) {if(eventSoundEnabledSetting==val) return;eventSoundEnabledSetting=val;QSettings().setValue("events/soundEnabled",val);emit eventAlertsChanged();}
    int eventAlertLeadMinutes() const {return qBound(1,eventAlertLeadMinutesSetting,60);}
    void setEventAlertLeadMinutes(int val) {val=qBound(1,val,60);if(eventAlertLeadMinutesSetting==val) return;eventAlertLeadMinutesSetting=val;QSettings().setValue("events/leadMinutes",val);emit eventAlertsChanged();}
    Q_INVOKABLE bool isEventTracked(const QString &eventId) const {
        return QSettings().value("events/track_"+eventId, true).toBool();
    }
    Q_INVOKABLE void setEventTracked(const QString &eventId, bool val) {
        QSettings().setValue("events/track_"+eventId, val);
        emit eventAlertsChanged();
    }
    Q_INVOKABLE void playAlertSound(const QString &soundType="event") {
        QString soundPath = (soundType=="boss") ? ":/sounds/boss_alert.wav" : ":/sounds/event_alert.wav";
        // Extract resource to temporary file if playing with external player
        QString tempWav = QDir::tempPath() + (soundType=="boss" ? "/slopmeter_boss.wav" : "/slopmeter_event.wav");
        if (!QFile::exists(tempWav)) {
            QFile res(soundPath);
            if (res.open(QIODevice::ReadOnly)) {
                QFile out(tempWav);
                if (out.open(QIODevice::WriteOnly)) {
                    out.write(res.readAll());
                    out.close();
                }
                res.close();
            }
        }
        // Play asynchronously
        QProcess::startDetached("canberra-gtk-play", {"-f", tempWav});
    }
    Q_INVOKABLE void dismissEventAlert() {
        currentActiveAlert.clear();
        emit eventAlertsChanged();
    }
    QVariantMap activeEventAlert() const {
        if (!currentActiveAlert.isEmpty() && QDateTime::currentDateTime() < currentActiveAlertUntil) {
            return currentActiveAlert;
        }
        return {};
    }
    static QDateTime calcNextEventOccurrence(const QString &id, const QDateTime &now) {
        if (id == "shugo") {
            QDateTime cand(now.date(), QTime(now.time().hour(), 0, 0));
            if (cand <= now) cand = cand.addSecs(3600);
            return cand;
        }
        if (id == "invasion") {
            QDateTime cand(now.date(), QTime(now.time().hour(), 30, 0));
            if (cand <= now) cand = cand.addSecs(3600);
            return cand;
        }
        if (id == "rift") {
            for (int h = 1; h <= 48; ++h) {
                QDateTime cand = now.addSecs(h * 3600);
                cand.setTime(QTime(cand.time().hour(), 0, 0));
                if (cand > now && (cand.time().hour() % 3 == 2)) return cand;
            }
        }
        if (id == "kaira") {
            for (int h = 1; h <= 48; ++h) {
                QDateTime cand = now.addSecs(h * 3600);
                cand.setTime(QTime(cand.time().hour(), 0, 0));
                if (cand > now && ((cand.time().hour() - 1 + 3) % 3 == 0)) return cand;
            }
        }
        if (id == "siege") {
            for (int d = 0; d < 14; ++d) {
                QDateTime cand(now.date().addDays(d), QTime(21, 0, 0));
                int dow = cand.date().dayOfWeek();
                if (cand > now && (dow == 1 || dow == 4 || dow == 6)) return cand;
            }
        }
        if (id == "siege_bosses") {
            for (int d = 0; d < 14; ++d) {
                QDateTime cand(now.date().addDays(d), QTime(21, 30, 0));
                int dow = cand.date().dayOfWeek();
                if (cand > now && (dow == 1 || dow == 4 || dow == 6)) return cand;
            }
        }
        if (id == "nahma") {
            for (int d = 0; d < 14; ++d) {
                QDateTime cand(now.date().addDays(d), QTime(21, 0, 0));
                int dow = cand.date().dayOfWeek();
                if (cand > now && (dow == 5 || dow == 7)) return cand;
            }
        }
        if (id == "daily_reset") {
            return QDateTime(now.date().addDays(1), QTime(0, 0, 0));
        }
        if (id == "weekly_reset") {
            for (int d = 1; d <= 7; ++d) {
                QDateTime cand(now.date().addDays(d), QTime(0, 0, 0));
                if (cand.date().dayOfWeek() == 3) return cand;
            }
        }
        return now.addSecs(3600);
    }
    QVariantList eventSchedule() const {
        struct EventDef {
            QString id;
            QString name;
            QString icon;
            QString category; // "minigame", "boss", "pvp", "reset"
            QString subtext;
            QString windowDuration;
        };
        static const QList<EventDef> defs = {
            {"shugo", "Festival de Shugo", "🎪", "minigame", "Minijuegos para 4 a 9 jugadores (Jump Jump, Wraith Evasion, Odyle Flight...)", "10m"},
            {"rift", "Spacetime Rift", "🌀", "pvp", "Apertura de portal dimensional a zona enemiga (portal abierto 10m)", "10m"},
            {"invasion", "Invasión Dimensional", "⚔️", "pvp", "Asalto Aéreo de Beritra, Espada Maldita o Asalto de Espíritus", "13m"},
            {"kaira", "Watcher Kaira", "👁️", "boss", "Spawn aleatorio en Lower Reshanta (América/Los Ángeles)", "30m"},
            {"siege", "Artifact Siege", "🏰", "pvp", "Acceso al Abyss Corridor para la facción ganadora (lun, jue, sáb)", "30m"},
            {"siege_bosses", "Siege Bosses", "👹", "boss", "Executors en Lower Reshanta y Executioners en Middle Reshanta", "30m"},
            {"nahma", "Guardian Lord Nahma", "👑", "boss", "World Boss en Lower y Middle Reshanta (vie, dom)", "30m"},
            {"daily_reset", "Reinicio Diario", "🌅", "reset", "Reinicio de misiones diarias, mazmorras y tiendas", "Instant"},
            {"weekly_reset", "Reinicio Semanal", "🔄", "reset", "Reinicio de misiones semanales y clasificaciones (miércoles)", "Instant"}
        };
        const QDateTime now = QDateTime::currentDateTime();
        QVariantList out;
        for (const auto &def : defs) {
            bool tracked = isEventTracked(def.id);
            QDateTime nextTime = calcNextEventOccurrence(def.id, now);
            qint64 secsLeft = now.secsTo(nextTime);
            if (secsLeft < 0) secsLeft = 0;
            int hours = secsLeft / 3600;
            int mins = (secsLeft % 3600) / 60;
            int secs = secsLeft % 60;
            QString timeStr;
            if (hours > 0) timeStr = QString("%1h %2m").arg(hours).arg(mins);
            else if (mins > 0) timeStr = QString("%1m %2s").arg(mins).arg(secs);
            else timeStr = QString("%1s").arg(secs);

            out.append(QVariantMap{
                {"id", def.id},
                {"name", def.name},
                {"icon", def.icon},
                {"category", def.category},
                {"subtext", def.subtext},
                {"windowDuration", def.windowDuration},
                {"startTime", nextTime.toString("HH:mm")},
                {"startDate", nextTime.toString("ddd, d MMM")},
                {"secsLeft", secsLeft},
                {"timeLeftStr", timeStr},
                {"tracked", tracked}
            });
        }
        // Sort by secsLeft ascending (closest upcoming events first)
        std::sort(out.begin(), out.end(), [](const QVariant &a, const QVariant &b) {
            return a.toMap().value("secsLeft").toLongLong() < b.toMap().value("secsLeft").toLongLong();
        });
        return out;
    }
    void checkEventAlerts() {
        if (!eventAlertsEnabledSetting) return;
        const QDateTime now = QDateTime::currentDateTime();
        const int leadSecs = eventAlertLeadMinutesSetting * 60;
        const auto events = eventSchedule();
        for (const auto &item : events) {
            const auto map = item.toMap();
            if (!map.value("tracked").toBool()) continue;
            const QString eventId = map.value("id").toString();
            const qint64 secsLeft = map.value("secsLeft").toLongLong();
            // Trigger alert when within lead time window (e.g. <= 5 min and > 0)
            if (secsLeft <= leadSecs && secsLeft > 0) {
                QString alertKey = QString("%1_%2").arg(eventId, map.value("startTime").toString());
                if (!triggeredEventAlerts.contains(alertKey)) {
                    triggeredEventAlerts.insert(alertKey);
                    currentActiveAlert = map;
                    currentActiveAlert["leadMinutes"] = eventAlertLeadMinutesSetting;
                    currentActiveAlertUntil = now.addSecs(180); // Keep banner visible for 3 minutes or until dismissed
                    if (eventSoundEnabledSetting) {
                        playAlertSound(map.value("category").toString() == "boss" ? "boss" : "event");
                    }
                    emit eventAlertsChanged();
                    break;
                }
            }
        }
        // Clean up stale triggers (more than 2 hours in the past)
        if (triggeredEventAlerts.size() > 50) triggeredEventAlerts.clear();
    }
    QNetworkAccessManager netMgr;
    QVariantList marketServerList;
    int selectedMarketServerId=QSettings().value("market/serverId", 11101).toInt();
    QString selectedMarketServerNameStr="Siel (NAE)";
    double marketTaxRateSetting=QSettings().value("market/taxRate", 10.0).toDouble();
    bool isMarketLoading=false;
    QString marketStatusMsg="Listo para consultar precios";
    QMap<int, double> marketPrices; // item_key -> unit price in Kinah
    QMap<int, int> marketListings; // item_key -> active listings count
    QVariantList cachedRecipes;
    bool recipesLoaded=false;

    double marketTaxRate() const {return marketTaxRateSetting;}
    void setMarketTaxRate(double val) {
        val = qBound(0.0, val, 30.0);
        if (qFuzzyCompare(marketTaxRateSetting, val)) return;
        marketTaxRateSetting = val;
        QSettings().setValue("market/taxRate", val);
        emit marketChanged();
    }

    QVariantList marketServers() const {
        if (!marketServerList.isEmpty()) return marketServerList;
        return {
            QVariantMap{{"id", 11101}, {"name", "Siel"}, {"region", "NAE"}, {"faction", "Elyos"}},
            QVariantMap{{"id", 12101}, {"name", "Israphel"}, {"region", "NAE"}, {"faction", "Asmodian"}},
            QVariantMap{{"id", 11102}, {"name", "Nezekan"}, {"region", "NAE"}, {"faction", "Elyos"}},
            QVariantMap{{"id", 12102}, {"name", "Zikel"}, {"region", "NAE"}, {"faction", "Asmodian"}},
            QVariantMap{{"id", 11201}, {"name", "Kaisinel"}, {"region", "NAW"}, {"faction", "Elyos"}},
            QVariantMap{{"id", 12201}, {"name", "Lumiel"}, {"region", "NAW"}, {"faction", "Asmodian"}},
            QVariantMap{{"id", 11301}, {"name", "Telemachus"}, {"region", "EU"}, {"faction", "Elyos"}},
            QVariantMap{{"id", 12301}, {"name", "Perento"}, {"region", "EU"}, {"faction", "Asmodian"}}
        };
    }
    int selectedMarketServer() const {return selectedMarketServerId;}
    QString selectedMarketServerName() const {return selectedMarketServerNameStr;}
    void setSelectedMarketServer(int id) {
        if (selectedMarketServerId == id) return;
        selectedMarketServerId = id;
        QSettings().setValue("market/serverId", id);
        for (const auto &s : marketServers()) {
            auto m = s.toMap();
            if (m.value("id").toInt() == id) {
                selectedMarketServerNameStr = QString("%1 (%2)").arg(m.value("name").toString(), m.value("region").toString());
                break;
            }
        }
        emit marketChanged();
        fetchMarketPrices();
    }
    bool marketLoading() const {return isMarketLoading;}
    QString marketStatus() const {return marketStatusMsg;}

    void loadRecipesDatabase() {
        if (recipesLoaded) return;
        QFile file(":/data/recipes.json");
        if (file.open(QIODevice::ReadOnly)) {
            auto doc = QJsonDocument::fromJson(file.readAll());
            if (doc.isArray()) {
                cachedRecipes = doc.array().toVariantList();
                recipesLoaded = true;
            }
        }
    }

    Q_INVOKABLE void fetchMarketPrices() {
        if (isMarketLoading) return;
        isMarketLoading = true;
        marketStatusMsg = "Consultando aion2.exchange...";
        emit marketChanged();

        // 1. Fetch server list dynamically if not populated
        if (marketServerList.isEmpty()) {
            QUrl srvUrl("https://aion2.exchange/api/servers");
            QNetworkRequest srvReq(srvUrl);
            srvReq.setHeader(QNetworkRequest::UserAgentHeader, "FullAIon2Companion/1.0");
            auto *srvReply = netMgr.get(srvReq);
            connect(srvReply, &QNetworkReply::finished, this, [this, srvReply]() {
                srvReply->deleteLater();
                if (srvReply->error() == QNetworkReply::NoError) {
                    auto doc = QJsonDocument::fromJson(srvReply->readAll());
                    if (doc.isObject()) {
                        auto arr = doc.object().value("servers").toArray();
                        QVariantList list;
                        for (const auto &sVal : arr) {
                            auto s = sVal.toObject();
                            if (s.value("has_data").toBool(true)) {
                                list.append(QVariantMap{
                                    {"id", s.value("server_id").toInt()},
                                    {"name", QString("%1 (%2 - %3)").arg(s.value("name").toString(), s.value("region").toString(), s.value("faction").toString())},
                                    {"rawName", s.value("name").toString()},
                                    {"region", s.value("region").toString()},
                                    {"faction", s.value("faction").toString()}
                                });
                            }
                        }
                        if (!list.isEmpty()) {
                            marketServerList = list;
                            emit marketChanged();
                        }
                    }
                }
            });
        }

        // 2. Fetch all items for current server
        QUrl url(QString("https://aion2.exchange/api/items?server=%1&limit=1000&sort=name&order=asc&window=24h").arg(selectedMarketServerId));
        QNetworkRequest req(url);
        req.setHeader(QNetworkRequest::UserAgentHeader, "FullAIon2Companion/1.0");

        QNetworkReply *reply = netMgr.get(req);
        connect(reply, &QNetworkReply::finished, this, [this, reply]() {
            isMarketLoading = false;
            reply->deleteLater();
            if (reply->error() != QNetworkReply::NoError) {
                marketStatusMsg = "Error al conectar con aion2.exchange: " + reply->errorString();
                emit marketChanged();
                return;
            }
            auto doc = QJsonDocument::fromJson(reply->readAll());
            if (!doc.isObject()) {
                marketStatusMsg = "Respuesta no válida del mercado";
                emit marketChanged();
                return;
            }
            auto items = doc.object().value("items").toArray();
            int countActive = 0, countLast = 0;
            for (const auto &it : items) {
                auto obj = it.toObject();
                int key = obj.value("item_key").toInt();
                int listingsCount = obj.value("listings").toInt(0);
                if (key > 0) {
                    marketListings[key] = listingsCount;
                }
                // Check active listing price first, fallback to last seen price
                double p = obj.value("price").toDouble(0.0);
                if (p > 0) {
                    marketPrices[key] = p;
                    countActive++;
                } else {
                    double lastP = obj.value("last_price").toDouble(0.0);
                    if (key > 0 && lastP > 0) {
                        marketPrices[key] = lastP;
                        countLast++;
                    }
                }
            }
            marketStatusMsg = QString("Mercado actualizado · %1 cotizaciones activas (%2 referenciales recientes)").arg(countActive).arg(countLast);
            emit marketChanged();
        });
    }

    QVariantList craftRecipes() {
        loadRecipesDatabase();
        QVariantList out;
        for (const auto &val : cachedRecipes) {
            auto r = val.toMap();
            auto res = r.value("result").toMap();
            int resultId = res.value("id").toInt();
            int resultCount = qMax(1, res.value("count").toInt());
            double sellPrice = marketPrices.value(resultId, 0.0);
            int resultListings = marketListings.value(resultId, 0);

            // Compute material costs
            auto mats = r.value("materials").toList();
            double totalCost = 0.0;
            bool missingAnyPrice = false;
            QVariantList enrichedMats;
            for (const auto &mVal : mats) {
                auto m = mVal.toMap();
                int mId = m.value("id").toInt();
                int mCount = qMax(1, m.value("count").toInt());
                double unitPrice = marketPrices.value(mId, 0.0);
                int matListings = marketListings.value(mId, 0);
                if (unitPrice <= 0) missingAnyPrice = true;
                double matCost = unitPrice * mCount;
                totalCost += matCost;
                m["unitPrice"] = unitPrice;
                m["totalCost"] = matCost;
                m["listings"] = matListings;
                enrichedMats.append(m);
            }

            // Broker tax in Aion 2 (Default is 10.0%, configurable by user)
            double taxMultiplier = 1.0 - (marketTaxRateSetting / 100.0);
            if (taxMultiplier < 0) taxMultiplier = 0;
            double netRevenue = sellPrice * resultCount * taxMultiplier;
            double profit = netRevenue - totalCost;
            double roiPct = totalCost > 0 ? (profit / totalCost * 100.0) : 0.0;

            res["listings"] = resultListings;
            r["result"] = res;
            r["materials"] = enrichedMats;
            r["sellPrice"] = std::round(sellPrice);
            r["totalCost"] = std::round(totalCost);
            r["profit"] = std::round(profit);
            r["roiPct"] = std::round(roiPct * 10.0) / 10.0;
            r["hasPrices"] = (sellPrice > 0 && !missingAnyPrice);
            r["resultListings"] = resultListings;

            out.append(r);
        }
        return out;
    }

    Q_INVOKABLE QVariantMap simulateBatchCraft(const QVariantMap &recipe, int batchCount) {
        batchCount = qBound(1, batchCount, 500);
        auto res = recipe.value("result").toMap();
        int resultCountPerCraft = qMax(1, res.value("count").toInt());
        double baseSellPrice = recipe.value("sellPrice").toDouble();
        int prodListings = recipe.value("resultListings").toInt();
        if (prodListings <= 0) prodListings = res.value("listings").toInt();
        if (prodListings <= 0) prodListings = 5;

        auto mats = recipe.value("materials").toList();

        double cumulativeCost = 0.0;
        double cumulativeGrossRevenue = 0.0;
        int maxProfitableBatch = 0;
        bool brokeEven = false;
        double taxMultiplier = 1.0 - (marketTaxRateSetting / 100.0);
        if (taxMultiplier < 0) taxMultiplier = 0.0;

        // Simulate unit by unit up to batchCount (and find break-even limit up to 100)
        int searchLimit = qMax(batchCount, 100);

        for (int i = 1; i <= searchLimit; ++i) {
            // Cost of crafting unit i
            double unitCraftCost = 0.0;
            for (const auto &mVal : mats) {
                auto m = mVal.toMap();
                double baseMatPrice = m.value("unitPrice").toDouble();
                int countPerCraft = qMax(1, m.value("count").toInt());
                int matListings = m.value("listings").toInt();
                if (matListings <= 0) matListings = 5;

                // Model order-book depth escalation:
                // Surge factor increases as more lots are consumed.
                double surgeFactor = 1.0 + (static_cast<double>(i - 1) / static_cast<double>(matListings)) * 0.15;
                if (surgeFactor > 5.0) surgeFactor = 5.0; // Cap surge at 500%
                unitCraftCost += (baseMatPrice * surgeFactor) * countPerCraft;
            }

            // Sale revenue of unit i
            // As more finished products flood the broker, price experiences mild saturation decay
            double decayFactor = 1.0 - (static_cast<double>(i - 1) / static_cast<double>(prodListings)) * 0.08;
            if (decayFactor < 0.4) decayFactor = 0.4; // Cap decay at floor 40% of base price
            double unitGrossRevenue = (baseSellPrice * decayFactor) * resultCountPerCraft;
            double unitNetRevenue = unitGrossRevenue * taxMultiplier;

            // Check if this single marginal craft is still profitable
            double marginalProfit = unitNetRevenue - unitCraftCost;
            if (marginalProfit > 0 && !brokeEven) {
                maxProfitableBatch = i;
            } else if (marginalProfit <= 0) {
                brokeEven = true;
            }

            if (i <= batchCount) {
                cumulativeCost += unitCraftCost;
                cumulativeGrossRevenue += unitGrossRevenue;
            }
        }

        double totalNetRevenue = cumulativeGrossRevenue * taxMultiplier;
        double totalProfit = totalNetRevenue - cumulativeCost;
        double totalRoi = cumulativeCost > 0 ? (totalProfit / cumulativeCost * 100.0) : 0.0;

        // Average unit values for the chosen batch
        double avgCostPerCraft = cumulativeCost / batchCount;
        double avgNetRevenuePerCraft = totalNetRevenue / batchCount;
        double avgProfitPerCraft = totalProfit / batchCount;

        return QVariantMap{
            {"batchCount", batchCount},
            {"totalCost", std::round(cumulativeCost)},
            {"totalGrossRevenue", std::round(cumulativeGrossRevenue)},
            {"totalNetRevenue", std::round(totalNetRevenue)},
            {"totalProfit", std::round(totalProfit)},
            {"totalRoi", std::round(totalRoi * 10.0) / 10.0},
            {"avgCostPerCraft", std::round(avgCostPerCraft)},
            {"avgNetRevenuePerCraft", std::round(avgNetRevenuePerCraft)},
            {"avgProfitPerCraft", std::round(avgProfitPerCraft)},
            {"maxProfitableBatch", maxProfitableBatch},
            {"isBatchProfitable", totalProfit > 0}
        };
    }
    QQuickWindow *window=nullptr;
    bool headless=false;
    QPoint dragOrigin;
    bool dragging=false, autoDisplay=true, detectionTest=false;
    QQuickItem *panel=nullptr;
    QString sceneLabel;
    QString encounterLabel="Waiting for combat";
    QVariantMap bossState;
    bool pvpEnabledSetting=QSettings().value("pvp/enabled",true).toBool();
    bool pvpEnabled() const {return pvpEnabledSetting;}
    void setPvpEnabled(bool value) {
        if(pvpEnabledSetting==value) return;
        pvpEnabledSetting=value;
        QSettings().setValue("pvp/enabled",value);
        emit pvpSettingChanged();
        emit pvpChanged();
    }
    bool pvpActive=false;
    QVariantMap pvpState;
    QVariantList radarRows;
    bool captureReady=false,replayMode=false;
    QString displayMessage="Waiting for AION 2 window…", gameScreen, scriptName;
    QTemporaryFile script;
    Q_INVOKABLE QString displayStatus() const {return displayMessage;}
    QString encounter() const {return encounterLabel;}
    QString sceneName() const {return sceneLabel;}
    QVariantMap boss() const {return bossState;}
    bool isPvP() const {return pvpEnabledSetting && pvpActive;}
    QVariantMap pvpSummary() const {return pvpState;}
    QVariantList displayRadar() const {
        if(!radarEnabledSetting) return {};
        QVariantList filtered;
        for(const auto &val:radarRows) {
            const auto m=val.toMap();
            const auto cat=m.value("category").toString();
            const auto name=m.value("name").toString();
            const bool catMatch =
                (cat=="chest" && radarChestsSetting) ||
                (cat=="odyle" && radarOdyleSetting) ||
                (cat=="gem" && radarGemsSetting) ||
                (cat=="ore" && radarOresSetting) ||
                (cat=="herb" && (radarHerbsSetting || radarResources())) ||
                (cat=="wood" && radarWoodSetting) ||
                (cat=="resource" && (radarHerbsSetting || radarResources())) ||
                (cat=="object" && radarOtherSetting);
            if(!catMatch) continue;
            if(!radarItemFilterSetting.isEmpty() && !name.contains(radarItemFilterSetting, Qt::CaseInsensitive)) continue;
            filtered.append(m);
        }
        return filtered;
    }
    QString healthMessage;
    QString bossHealthStatus() const {return previewEnabled?"Test mode: simulated boss HP.":healthMessage;}
    bool captureActive() const {return captureReady;}
    void setCaptureReady(bool ready) {if(captureReady!=ready) {
        captureReady=ready;emit captureChanged();
        if(ready) {interfaceMessage="Capturing on "+(activeInterface.isEmpty()?"Automatic":activeInterface)+". Relog if no character is detected.";emit interfacesChanged();}
    }}
    QString displayLabel() const {auto name=window&&window->screen()?window->screen()->name():QString();return "Display - "+(name.isEmpty()?"Default":name);}
    QVariantList availableScreens() const {
        QVariantList out;
        for (auto *s : QGuiApplication::screens()) {
            if (!s) continue;
            auto geo = s->geometry();
            QString label = QString("%1 (%2x%3)").arg(s->name()).arg(geo.width()).arg(geo.height());
            if (s == QGuiApplication::primaryScreen()) label += " [Principal]";
            out.append(QVariantMap{
                {"name", s->name()},
                {"label", label},
                {"width", geo.width()},
                {"height", geo.height()},
                {"isPrimary", s == QGuiApplication::primaryScreen()}
            });
        }
        return out;
    }
    QString selectedScreenName() const {
        return window && window->screen() ? window->screen()->name() : "";
    }
    void setSelectedScreenName(const QString &name) {
        if (name.isEmpty()) return;
        autoDisplay = false;
        QSettings().setValue("display/autoFollow", false);
        QSettings().setValue("display/manualScreen", name);
        for (auto *s : QGuiApplication::screens()) {
            if (s && s->name() == name) {
                selectScreen(s);
                break;
            }
        }
        emit displayChanged();
    }
    bool followGameDisplay() const {
        return autoDisplay;
    }
    void setFollowGameDisplay(bool follow) {
        autoDisplay = follow;
        QSettings().setValue("display/autoFollow", follow);
        if (follow && !gameScreen.isEmpty()) {
            gameOutput(gameScreen, {});
        }
        emit displayChanged();
    }
    void updateInputMask() {
        if(window&&panel) {
            // An empty QWindow mask resets the region. Use an off-surface region
            // so an entirely invisible meter lets all game clicks through.
            window->setMask(panel->opacity()>0?QRegion(QRectF(panel->x(),panel->y(),panel->width(),panel->height()).toAlignedRect()):QRegion(QRect(-1,-1,1,1)));
        }
    }
    void resizeCanvas() {if(window) {window->resize(window->screen()->geometry().size());updateInputMask();}}
    void configureOverlay() {
        if(headless||!window) return;
        if(g_noLayerShell) {
            window->setFlags(Qt::Window|Qt::FramelessWindowHint|Qt::WindowStaysOnTopHint|Qt::Tool|Qt::WindowDoesNotAcceptFocus|Qt::X11BypassWindowManagerHint);
            window->setPosition(window->screen()->geometry().topLeft());
            if(!x11RaiseTimer.isActive()) {
                x11RaiseTimer.setInterval(1000);
                connect(&x11RaiseTimer,&QTimer::timeout,this,[this] {if(window&&window->isVisible()) window->raise();});
                x11RaiseTimer.start();
            }
            return;
        }
        auto layer=LayerShellQt::Window::get(window);
        layer->setLayer(LayerShellQt::Window::LayerOverlay);
        layer->setAnchors(LayerShellQt::Window::Anchors(LayerShellQt::Window::AnchorTop)|LayerShellQt::Window::AnchorBottom|LayerShellQt::Window::AnchorLeft|LayerShellQt::Window::AnchorRight);
#ifdef SLOPMETER_LAYER_DESIRED_SIZE
        layer->setDesiredSize(QSize(0,0));
#endif
#ifdef SLOPMETER_LAYER_SCREEN
        layer->setScreen(window->screen());
#endif
        layer->setMargins(QMargins());layer->setCloseOnDismissed(false);
        layer->setExclusiveZone(-1);layer->setKeyboardInteractivity(LayerShellQt::Window::KeyboardInteractivityNone);
    }
    void selectScreen(QScreen *screen) {
        if(!screen||!window||dragging) return;
        if(window->screen()!=screen) {
            const bool visible=window->isVisible();const auto pos=position();
            window->hide();
            // LayerShellQt 6.3 attaches its integration to the platform handle only
            // at construction. Recreate that attachment after destroying the handle.
            if(!headless&&!g_noLayerShell) delete LayerShellQt::Window::get(window);
            window->destroy();window->setScreen(screen);window->setPosition(screen->geometry().topLeft());resizeCanvas();
            configureOverlay();positionOverlay(pos);if(visible) window->show();
        }
        displayMessage=QString("%1 · %2").arg(autoDisplay?"Game display":"Manual display",screen->name());emit displayChanged();
    }
    void startGameDetection() {
        auto bus=QDBusConnection::sessionBus();bus.registerObject("/AionDPS",this,QDBusConnection::ExportAllSlots);
        QFile source(":/game-screen.js");if(!source.open(QIODevice::ReadOnly)||!script.open()) return;
        auto code=source.readAll();code.replace("@SERVICE@",bus.baseService().toUtf8());script.write(code);script.flush();
        scriptName=QString("aiondps-%1").arg(QCoreApplication::applicationPid());
        QDBusInterface scripting("org.kde.KWin","/Scripting","org.kde.kwin.Scripting",bus);
        auto watcher=new QDBusPendingCallWatcher(scripting.asyncCall("loadScript",script.fileName(),scriptName),this);
        connect(watcher,&QDBusPendingCallWatcher::finished,this,[this](QDBusPendingCallWatcher *call) {
            QDBusPendingReply<int> reply=*call;call->deleteLater();
            if(reply.isError()||reply.value()<0) {displayMessage="Automatic detection unavailable; choose a display in the tray.";emit displayChanged();return;}
            QDBusInterface running("org.kde.KWin",QString("/Scripting/Script%1").arg(reply.value()),"org.kde.kwin.Script",QDBusConnection::sessionBus());
            auto run=new QDBusPendingCallWatcher(running.asyncCall("run"),this);
            connect(run,&QDBusPendingCallWatcher::finished,this,[this](QDBusPendingCallWatcher *call) {
                if(call->isError()) {displayMessage="Game detection failed; choose a display in the tray.";emit displayChanged();}call->deleteLater();
            });
        });
    }
public slots:
    void gameOutput(const QString &output,const QString &title) {
        gameScreen=output;
        if(detectionTest) {
            fprintf(stderr,"KWin detection report: game output=%s title=%s\n",qPrintable(output),qPrintable(title));
            for(auto screen:QGuiApplication::screens()) fprintf(stderr,"Qt output: %s\n",qPrintable(screen->name()));
            QCoreApplication::exit(output.isEmpty()?6:0);return;
        }
        if(!autoDisplay) return;
        if(output.isEmpty()) {displayMessage="AION 2 window not found · select display in tray";emit displayChanged();return;}
        for(auto screen:QGuiApplication::screens()) if(screen->name()==output) {selectScreen(screen);return;}
    }
public:

    QVariantMap currentState,displayState;
    QVariantList history;
    quint64 selectedSession=0;
    quint32 selectedActor=0;
    QVariantList fightChoices() const {
        if(previewEnabled) return {QVariantMap{{"id",qulonglong(0)},{"label","Test mode"}}};
        QVariantList choices{QVariantMap{{"id",qulonglong(0)},{"label","Current fight"}}};
        for(const auto &value:history) {
            auto entry=value.toMap();auto state=entry.value("snapshot").toMap();auto boss=state.value("boss").toMap();
            auto label=boss.value("name").toString();
            if(state.value("pvp").toBool()) {
                auto opp=state.value("pvpSummary").toMap().value("opponentName").toString();
                label=opp.isEmpty()?"PvP":"PvP vs "+opp;
            }
            if(label.isEmpty()) label=state.value("scene").toMap().value("name").toString();
            if(label.isEmpty()) label="Combat";
            auto started=QDateTime::fromString(entry.value("started").toString(),Qt::ISODateWithMs).toLocalTime();
            choices.append(QVariantMap{{"id",state.value("session")},{"label",QString("#%1 · %2 · %3s · %4").arg(state.value("session").toULongLong()).arg(label).arg(state.value("duration").toDouble(),0,'f',1).arg(started.toString("MMM d HH:mm:ss"))}});
        }
        return choices;
    }
    int selectedFightIndex() const {
        if(previewEnabled) return 0;
        auto choices=fightChoices();for(int i=0;i<choices.size();++i) if(choices[i].toMap().value("id").toULongLong()==selectedSession) return i;return 0;
    }
    QVariantList displayPlayers() const {return displayState.value("actors").toList();}
    QVariantList displayEnemies() const {return isPvP() ? displayState.value("enemies").toList() : QVariantList{};}
    QVariantList detailChoices() const {
        QVariantList all=displayPlayers();
        for(const auto &enemy:displayEnemies()) {
            auto m=enemy.toMap();
            m["name"]=m.value("name").toString()+" (Rival)";
            all.append(m);
        }
        return all;
    }
    QVariantMap detailFight() const {return displayState;}
    QVariantMap detailPlayer() const {
        auto rows=displayPlayers();for(const auto &row:rows) if(row.toMap().value("id").toUInt()==selectedActor) return row.toMap();
        auto enemyRows=displayEnemies();for(const auto &row:enemyRows) if(row.toMap().value("id").toUInt()==selectedActor) return row.toMap();
        if(!rows.isEmpty()) return rows.first().toMap();
        return enemyRows.isEmpty()?QVariantMap{}:enemyRows.first().toMap();
    }
    Q_INVOKABLE void selectPlayer(quint32 id) {selectedActor=id;updateDetail();}
    Q_INVOKABLE void selectFight(quint64 id) {if(previewEnabled) return;selectedSession=id;refreshDisplay();emit historyChanged();}
    Q_INVOKABLE QString classIcon(const QString &name) const {
        const QStringList supplied{"Gladiator","Templar","Ranger","Assassin","Sorcerer","Cleric","Chanter","Elementalist","Brawler"};
        return supplied.contains(name)?"qrc:/class-icons/"+name+".png":"qrc:/icons/Unknown.svg";
    }
    void refreshDisplay() {
        displayState=previewEnabled?previewState():currentState;
        if(selectedSession&&!previewEnabled) {
            bool found=false;
            for(const auto &value:history) {auto entry=value.toMap();auto state=entry.value("snapshot").toMap();if(state.value("session").toULongLong()==selectedSession) {displayState=state;displayState["started"]=entry.value("started");displayState["ended"]=entry.value("ended");found=true;break;}}
            if(!found) {selectedSession=0;emit historyChanged();}
        }
        // Keep the last completed fight on Current fight until positive damage
        // starts a new session. History owns immutable names, skills and target HP.
        if(!previewEnabled&&!selectedSession&&!currentState.value("cleared").toBool()&&!currentState.value("active").toBool()&&currentState.value("actors").toList().isEmpty()&&currentState.value("enemies").toList().isEmpty()) {
            for(const auto &value:history) {auto state=value.toMap().value("snapshot").toMap();if(state.value("session")==currentState.value("session")) {displayState=state;break;}}
        }
        auto next=(!previewEnabled&&!selectedSession?currentState:displayState).value("character").toString();
        if(!previewEnabled&&!selectedSession&&!replayMode) {
            if(next.isEmpty()) next=rememberedName;
        }
        if(next!=name) {name=next;emit characterChanged();}
        double seconds=displayState.value("duration").toDouble();if(seconds!=span) {span=seconds;emit durationChanged();}
        auto label=displayState.value("encounter","Waiting for combat").toString();
        auto scene=displayState.value("scene").toMap().value("name").toString();
        if(label!=encounterLabel||scene!=sceneLabel) {encounterLabel=label;sceneLabel=scene;emit encounterChanged();}
        auto boss=displayState.value("boss").toMap();auto health=displayState.value("healthStatus").toString();if(boss!=bossState||health!=healthMessage) {bossState=boss;healthMessage=health;emit bossChanged();}
        const bool nextPvp=displayState.value("pvp").toBool();
        const auto nextPvpSummary=displayState.value("pvpSummary").toMap();
        if(nextPvp!=pvpActive||nextPvpSummary!=pvpState) {pvpActive=nextPvp;pvpState=nextPvpSummary;emit pvpChanged();}
        const auto nextRadar=(!previewEnabled&&!selectedSession?currentState:displayState).value("radar").toList();
        if(nextRadar!=radarRows) {radarRows=nextRadar;emit radarChanged();}
        actors.update(QJsonArray::fromVariantList(displayPlayers()));
        enemies.update(QJsonArray::fromVariantList(displayEnemies()));
        updateDetail();
    }
    void updateDetail() {skills.update(QJsonArray::fromVariantList(detailPlayer().value("skills").toList()));emit detailChanged();}
    QString appVersion() const {return SLOPMETER_VERSION;}
    QProcess interfaceLister;
    QString interfaceSetting=QSettings().value("capture/interface", "").toString();
    QVariantList interfaceRows{{QVariantMap{{"name",""},{"label","Automatic (capture default)"}}}};
    QString interfaceMessage;
    QStringList captureArguments;
    bool restartingCapture=false;
    QString activeInterface;
    QString captureInterface() const {return interfaceSetting;}
    QVariantList captureInterfaces() const {return interfaceRows;}
    bool captureReplay() const {return replayMode;}
    QString interfaceStatus() const {return interfaceMessage;}
    void setCaptureInterface(const QString &value) {
        if(replayMode||value==interfaceSetting) return;
        interfaceSetting=value;QSettings().setValue("capture/interface",value);
        interfaceMessage="Selection saved. Apply to restart capture.";emit interfacesChanged();
    }
    Q_INVOKABLE void refreshInterfaces() {
        if(interfaceLister.state()!=QProcess::NotRunning||backendPath.isEmpty()) return;
        interfaceLister.start(backendPath,{"-interfaces","-json"});
    }
    void startCapture() {
        auto arguments=captureArguments;
        activeInterface=interfaceSetting;
        if(!replayMode&&!activeInterface.isEmpty()) arguments<<"-interface"<<activeInterface;
        buffer.clear();pending={};setStatus(replayMode?"Starting replay…":"Starting capture…");process.start(backendPath,arguments);
    }
    Q_INVOKABLE void applyCaptureInterface() {
        if(replayMode||restartingCapture) return;
        if(process.state()!=QProcess::NotRunning&&interfaceSetting==activeInterface) {
            interfaceMessage="This interface is already active. Relog if no character is detected.";emit interfacesChanged();return;
        }
        interfaceMessage="Restarting capture… Relog your character after capture starts.";emit interfacesChanged();
        setCaptureReady(false);
        if(process.state()==QProcess::NotRunning) {startCapture();return;}
        restartingCapture=true;process.terminate();
        QTimer::singleShot(2000,this,[this] {if(restartingCapture&&process.state()!=QProcess::NotRunning) process.kill();});
    }
    QString backendPath;
    QString rememberedName=QSettings().value("capture/characterName", "").toString();
    QString characterName() const {return rememberedName;}
    void sendCharacterName() {
        if(!replayMode && process.state()!=QProcess::NotRunning) process.write(QJsonDocument(QJsonObject{{"characterName",rememberedName}}).toJson(QJsonDocument::Compact)+"\n");
    }
    void setCharacterName(const QString &value) {
        auto next=value.trimmed();if(next.toUtf8().size()>72) return;
        for(auto c:next) if(c.isSpace() && c!=' ') return;
        if(next==rememberedName) return;
        rememberedName=next;QSettings().setValue("capture/characterName",next);emit characterNameChanged();
        sendCharacterName();refreshDisplay();
    }
    QString name=rememberedName, message="Starting capture…";
    double span=0;
    Bridge() {
        connect(&interfaceLister,&QProcess::errorOccurred,this,[this] {interfaceMessage=interfaceLister.errorString();emit interfacesChanged();});
        connect(&interfaceLister,qOverload<int,QProcess::ExitStatus>(&QProcess::finished),this,[this](int code,QProcess::ExitStatus) {
            if(code==0) {
                auto document=QJsonDocument::fromJson(interfaceLister.readAllStandardOutput());
                if(document.isArray()) {
                    interfaceRows=document.array().toVariantList();bool found=false;
                    for(const auto &row:interfaceRows) if(row.toMap().value("name").toString()==interfaceSetting) found=true;
                    if(!found) interfaceRows.append(QVariantMap{{"name",interfaceSetting},{"label",interfaceSetting+" (unavailable)"}});
                }
            } else interfaceMessage=QString::fromUtf8(interfaceLister.readAllStandardError()).trimmed();
            emit interfacesChanged();
        });
        QFile preview(":/preview.json");if(preview.open(QIODevice::ReadOnly)) previewParty=QJsonDocument::fromJson(preview.readAll()).array().toVariantList();
        publish.setInterval(pollInterval());
        connect(&publish,&QTimer::timeout,this,[this] {
            if(previewEnabled) refreshDisplay();
        });
        eventTimer.setInterval(1000);
        connect(&eventTimer,&QTimer::timeout,this,[this] {
            checkEventAlerts();
            emit eventAlertsChanged();
        });
        eventTimer.start();
        connect(&process,&QProcess::started,this,[this] {sendRefreshInterval();sendCharacterName();});
        connect(&process,&QProcess::readyReadStandardOutput,this,[this] {
            buffer+=process.readAllStandardOutput();int end;
            while((end=buffer.indexOf('\n'))>=0) {
                auto line=buffer.left(end);buffer.remove(0,end+1);
                auto doc=QJsonDocument::fromJson(line);if(doc.isObject()) {
                    auto state=doc.object();
                    const auto detected=state.value("character").toString();
                    if(!replayMode&&!detected.isEmpty()&&detected!=rememberedName) {rememberedName=detected;QSettings().setValue("capture/characterName",detected);emit characterNameChanged();}
                    // History arrives only when changed. Preserve it even if coalescing drops a snapshot.
                    if(state.contains("history")) {history=state.value("history").toArray().toVariantList();emit historyChanged();}
                    pending=state;setCaptureReady(process.state()==QProcess::Running&&!replayMode);if(process.state()!=QProcess::NotRunning) setStatus(replayMode?"Replay running":state.value("character").toString().isEmpty()?"Capture active · waiting for character. Log out and back in with capture running.":"Capture running");}
            }
            // The backend already limits snapshots to the selected interval.
            // Apply the newest complete frame once per read, avoiding a second timer's latency.
            if(!pending.isEmpty()) {currentState=pending.toVariantMap();pending={};if(!previewEnabled) refreshDisplay();}
        });
        connect(&process,&QProcess::readyReadStandardError,this,[this] {auto text=QString::fromUtf8(process.readAllStandardError()).trimmed();if(!text.isEmpty()) setStatus(text);});
        connect(&process,&QProcess::errorOccurred,this,[this] {setCaptureReady(false);setStatus(process.errorString());});
        connect(&process,qOverload<int,QProcess::ExitStatus>(&QProcess::finished),this,[this](int code,QProcess::ExitStatus) {setCaptureReady(false);if(restartingCapture) {restartingCapture=false;QTimer::singleShot(0,this,[this]{startCapture();});return;}setStatus(code==0?"Capture finished":"Capture failed: "+message);});
    }
    ~Bridge() {
        restartingCapture=false;
        if(interfaceLister.state()!=QProcess::NotRunning) {interfaceLister.kill();interfaceLister.waitForFinished(1000);}
        if(!scriptName.isEmpty()) {QDBusInterface scripting("org.kde.KWin","/Scripting","org.kde.kwin.Scripting",QDBusConnection::sessionBus());scripting.asyncCall("unloadScript",scriptName);}
        if(process.state()!=QProcess::NotRunning) {process.terminate();if(!process.waitForFinished(1500)) {process.kill();process.waitForFinished();}}}
    QString character() const {return name;}
    QString status() const {return message;}
    double duration() const {return span;}
    void setStatus(const QString &text) {if(message!=text) {message=text;emit statusChanged();}}
    QPoint position() const {return panel?QPoint(qRound(panel->x()),qRound(panel->y())):QPoint();}
    void positionOverlay(QPoint pos) {
        if(!panel) return;
        pos.setX(qBound(0,pos.x(),qMax(0,window->width()-qRound(panel->width()))));
        pos.setY(qBound(0,pos.y(),qMax(0,window->height()-qRound(panel->height()))));
        panel->setPosition(pos);updateInputMask();
    }
    Q_INVOKABLE void beginDrag() {dragOrigin=position();dragging=true;}
    Q_INVOKABLE void dragOverlay(int dx,int dy) {positionOverlay(dragOrigin+QPoint(dx,dy));}
    Q_INVOKABLE void endDrag() {dragging=false;savePosition();if(autoDisplay&&!gameScreen.isEmpty()) gameOutput(gameScreen,{});}
    void savePosition() {QSettings().setValue("position0",position());if(panel) QSettings().setValue("positionBottom",panel->y()+panel->height());}
    Q_INVOKABLE void resetPosition() {positionOverlay({40,100});savePosition();window->show();}
    std::function<void()> hiddenRecovery;
    Q_INVOKABLE void hide() {window->hide();if(hiddenRecovery) hiddenRecovery();}
    Q_INVOKABLE void quit() {QCoreApplication::quit();}
signals:
    void interfacesChanged();
    void characterNameChanged();void captureChanged();void testModeChanged();void pollingChanged();void historyChanged();void detailChanged();void appearanceChanged();void characterChanged();void statusChanged();void durationChanged();void displayChanged();void encounterChanged();void bossChanged();void pvpChanged();void pvpSettingChanged();void radarChanged();
    void radarSelectionChanged();void radarRangeChanged();void radarSubmenuChanged();void eventAlertsChanged();void marketChanged();
};
static void global(void *data,wl_registry*,uint32_t,const char *interface,uint32_t) {if(!std::strcmp(interface,"zwlr_layer_shell_v1")) *static_cast<bool*>(data)=true;}
static void removed(void*,wl_registry*,uint32_t) {}
int main(int argc,char **argv) {
    for(int i=1;i<argc;++i) if(std::strcmp(argv[i],"--version")==0) {printf("SlopMeter %s\n",SLOPMETER_VERSION);return 0;}

    bool headless=qEnvironmentVariable("QT_QPA_PLATFORM")=="offscreen";
    if(!headless) {
        auto display=wl_display_connect(nullptr);if(!display) {fprintf(stderr,"A Wayland session is required.\n");return 1;}
        bool supported=false;auto registry=wl_display_get_registry(display);const wl_registry_listener listener{global,removed};
        wl_registry_add_listener(registry,&listener,&supported);wl_display_roundtrip(display);wl_registry_destroy(registry);wl_display_disconnect(display);
        if(!supported) {if(qEnvironmentVariableIntValue("SLOPMETER_X11_FALLBACK")!=1) {fprintf(stderr,"Wayland layer shell is required (set SLOPMETER_X11_FALLBACK=1 to try an XWayland fallback).\n");return 1;}fprintf(stderr,"Layer shell not available: using XWayland fallback.\n");g_noLayerShell=true;qputenv("QT_QPA_PLATFORM","xcb");} else qputenv("QT_QPA_PLATFORM","wayland");
    }
    qunsetenv("QT_WAYLAND_SHELL_INTEGRATION");
    QApplication app(argc,argv);app.setQuitOnLastWindowClosed(false);app.setOrganizationName("AionDPS");app.setApplicationName("SlopMeter");
    const QIcon appIcon(":/app/slopmeter.svg");app.setWindowIcon(appIcon);
    const bool isolated=app.arguments().contains("--ui-self-test")||app.arguments().contains("--screenshots")||app.arguments().contains("--ui-detection-test")||headless;
    if(!isolated) {
        if(registerDesktopIdentity()) app.setDesktopFileName("slopmeter");
        else fprintf(stderr,"Desktop registration failed: the user data directory is not writable.\n");
    }
    const QString desktopService="io.github.Netskill89.SlopMeter";
    auto desktopBus=QDBusConnection::sessionBus();DesktopControl desktopControl;
    if(!isolated&&desktopBus.isConnected()&&!desktopBus.registerService(desktopService)) {
        QDBusInterface existing(desktopService,"/SlopMeter",desktopService,desktopBus);
        auto reply=existing.call("Show");
        if(reply.type()!=QDBusMessage::ErrorMessage) return 0;
        fprintf(stderr,"Could not restore the existing SlopMeter instance: %s\n",qPrintable(reply.errorMessage()));return 1;
    }
    if(!isolated&&desktopBus.isConnected()) desktopBus.registerObject("/SlopMeter",&desktopControl,QDBusConnection::ExportAllSlots);
    std::unique_ptr<QTemporaryDir> selfTestSettings;
    if(app.arguments().contains("--ui-self-test")||app.arguments().contains("--screenshots")) {
        selfTestSettings=std::make_unique<QTemporaryDir>();if(!selfTestSettings->isValid()) return 1;
        QSettings::setDefaultFormat(QSettings::IniFormat);
        QSettings::setPath(QSettings::IniFormat,QSettings::UserScope,selfTestSettings->path());
    }
    if(!selfTestSettings) {
        QSettings previous("AionDPS","AionDPS"), current;
        if(!current.value("migration/aionDPS",false).toBool()) {
            for(const auto &key:previous.allKeys()) if(!current.contains(key)) current.setValue(key,previous.value(key));
            current.setValue("migration/aionDPS",true);
        }
    }
    Bridge bridge;bridge.headless=headless;
    ReleaseCheck updates(SLOPMETER_VERSION);
    QQmlApplicationEngine engine;engine.rootContext()->setContextProperty("backend",&bridge);engine.rootContext()->setContextProperty("actorModel",&bridge.actors);engine.rootContext()->setContextProperty("enemyModel",&bridge.enemies);engine.rootContext()->setContextProperty("skillModel",&bridge.skills);
    engine.rootContext()->setContextProperty("updates",&updates);
    engine.load(QUrl("qrc:/Main.qml"));if(engine.rootObjects().size()!=1) return 1;
    auto window=qobject_cast<QQuickWindow*>(engine.rootObjects().first());if(!window) return 1;bridge.window=window;
    bridge.panel=window->findChild<QQuickItem*>("meterPanel");if(!bridge.panel) return 1;
    QObject::connect(bridge.panel,&QQuickItem::xChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::yChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::opacityChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::widthChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(bridge.panel,&QQuickItem::heightChanged,&bridge,[&bridge, previousHeight=bridge.panel->height()]() mutable {
        const auto height=bridge.panel->height();auto pos=bridge.position();
        if(bridge.growUp()) pos.ry()+=qRound(previousHeight-height);
        previousHeight=height;bridge.positionOverlay(pos);
    });
    QStringList args=app.arguments().mid(1);bool test=args.removeAll("--ui-self-test")>0;
    QString screenshotDirectory;
    auto screenshotOption=args.indexOf("--screenshots");
    if(screenshotOption>=0) {if(screenshotOption+1>=args.size()) {fprintf(stderr,"--screenshots needs an output directory.\n");return 1;}screenshotDirectory=args.takeAt(screenshotOption+1);args.removeAt(screenshotOption);}
    bridge.detectionTest=args.removeAll("--ui-detection-test")>0;
    if(bridge.detectionTest) {
        bridge.startGameDetection();QTimer::singleShot(4000,&app,[&app] {fprintf(stderr,"Game-detection callback timed out.\n");app.exit(7);});return app.exec();
    }
    bridge.configureOverlay();
    QObject::connect(window,&QQuickWindow::widthChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(window,&QQuickWindow::heightChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(window,&QWindow::screenChanged,&bridge,[&bridge](QScreen*) {emit bridge.displayChanged();});
    
    // Restore saved screen preference if user had selected a specific monitor
    bool autoFollow = QSettings().value("display/autoFollow", true).toBool();
    QString manualScreen = QSettings().value("display/manualScreen", "").toString();
    bridge.autoDisplay = autoFollow;
    if (!autoFollow && !manualScreen.isEmpty()) {
        for (auto *s : QGuiApplication::screens()) {
            if (s && s->name() == manualScreen) {
                bridge.selectScreen(s);
                break;
            }
        }
    }

    bridge.resizeCanvas();emit bridge.displayChanged();
    auto savedPosition=QSettings().value("position0",QPoint(40,100)).toPoint();
    if(bridge.growUp() && QSettings().contains("positionBottom")) savedPosition.setY(qRound(QSettings().value("positionBottom").toDouble()-bridge.panel->height()));
    bridge.positionOverlay(savedPosition);window->show();
    QSystemTrayIcon tray{appIcon};tray.setToolTip("SlopMeter");QMenu menu;
    menu.addAction("Show meter",window,[window] {window->show();});menu.addAction("Hide meter",&bridge,&Bridge::hide);
    auto displays=menu.addMenu("Display");
    displays->addAction("Follow AION 2",&bridge,[&bridge] {bridge.autoDisplay=true;bridge.gameOutput(bridge.gameScreen,{});});
    for(auto screen:app.screens()) {const auto name=screen->name();displays->addAction(name,&bridge,[&bridge,name] {bridge.autoDisplay=false;for(auto screen:QGuiApplication::screens()) if(screen->name()==name) {bridge.selectScreen(screen);break;}});}
    menu.addAction("Settings",window,[window] {QMetaObject::invokeMethod(window,"openSettings");});
    menu.addAction("Reset position",&bridge,&Bridge::resetPosition);menu.addSeparator();menu.addAction("Quit",&bridge,&Bridge::quit);
    tray.setContextMenu(&menu);QObject::connect(&tray,&QSystemTrayIcon::activated,&bridge,[window](QSystemTrayIcon::ActivationReason r) {if(r==QSystemTrayIcon::Trigger) window->setVisible(!window->isVisible());});tray.show();
    // Desktops without a tray must still provide a visible way back to the meter.
    QWidget recovery;recovery.setWindowTitle("SlopMeter");recovery.setWindowIcon(appIcon);
    auto recoveryLayout=new QVBoxLayout(&recovery);
    recoveryLayout->addWidget(new QLabel("SlopMeter is running. Your desktop tray is unavailable.",&recovery));
    auto restoreButton=new QPushButton("Show meter",&recovery);recoveryLayout->addWidget(restoreButton);
    auto restore=[&] {window->show();recovery.hide();};
    desktopControl.meterVisible=[window] {return window->isVisible();};
    QObject::connect(restoreButton,&QPushButton::clicked,&app,restore);
    QObject::connect(&desktopControl,&DesktopControl::restoreRequested,&app,restore);
    bridge.hiddenRecovery=[&] {if(!headless&&!QSystemTrayIcon::isSystemTrayAvailable()) recovery.show();};
    fprintf(stderr,"Desktop tray: %s. Launch SlopMeter again to restore the meter.\n",QSystemTrayIcon::isSystemTrayAvailable()?"available":"unavailable");
    if(!test||!headless) bridge.startGameDetection();
    auto executable=qEnvironmentVariable("SLOPMETER_BACKEND");
    if(executable.isEmpty()) executable=qEnvironmentVariable("AIONDPS_BACKEND");
    if(executable.isEmpty()) {
        executable=app.applicationDirPath()+"/slopmeter-capture";
        if(!QFile::exists(executable)) executable=app.applicationDirPath()+"/../slopmeter-capture";
    }
    bridge.backendPath=executable;
    for(const auto &arg:args) if(arg=="-read"||arg=="--read"||arg.startsWith("-read=")||arg.startsWith("--read=")) bridge.replayMode=true;
    if(!screenshotDirectory.isEmpty()) {
        if(headless) {fprintf(stderr,"Documentation screenshots require a real Wayland desktop.\n");return 1;}
        bridge.setBarWidth(440);bridge.setBarHeight(44);bridge.setBarSpacing(6);bridge.setBackgroundOpacity(100);bridge.setTestMode(true);
        QDir().mkpath(screenshotDirectory);
        QTimer::singleShot(1400,&app,[&,screenshotDirectory] {
            auto grab=bridge.panel->grabToImage();
            QObject::connect(grab.get(),&QQuickItemGrabResult::ready,&app,[&,grab,screenshotDirectory] {
                if(!grab->image().save(screenshotDirectory+"/meter.png")) {app.exit(46);return;}
                const auto actor=bridge.displayPlayers().first().toMap().value("id").toUInt();
                QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)));
                QTimer::singleShot(700,&app,[&,screenshotDirectory] {
                    for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="fightDetailsWindow") {
                        auto details=qobject_cast<QQuickWindow*>(candidate);details->resize(1400,740);
                        QTimer::singleShot(500,&app,[&,details,screenshotDirectory] {
                            if(!details->grabWindow().save(screenshotDirectory+"/damage-breakdown.png")) {app.exit(47);return;}
                            fprintf(stderr,"Native Wayland screenshots saved.\n");app.quit();
                        });return;
                    }
                    app.exit(48);
                });
            });
        });
        return app.exec();
    }
    args.prepend("-json");args.append("-interval");args.append(QString::number(bridge.pollInterval())+"ms");for(int i=0;i<args.size();++i) if(args[i]=="-interface"&&i+1<args.size()) {bridge.setCaptureInterface(args[i+1]);args.removeAt(i+1);args.removeAt(i);break;}
    bridge.captureArguments=args;bridge.startCapture();bridge.refreshInterfaces();
    if(!isolated) updates.start();
    if(test) QTimer::singleShot(500,&app,[&] {
        if(QGuiApplication::allWindows().size()!=1) {app.exit(2);return;}
        bridge.hide();window->show();if(!window->isVisible()) {app.exit(3);return;}
        window->hide();desktopControl.Show();if(!window->isVisible()||recovery.isVisible()) {app.exit(63);return;}
        if(appIcon.isNull()||appIcon.pixmap(32,32).isNull()||!tray.isVisible()) {app.exit(64);return;}
        ActorModel model;
        int resets=0,insertions=0;
        QObject::connect(&model,&QAbstractItemModel::modelReset,&app,[&] {++resets;});
        QObject::connect(&model,&QAbstractItemModel::rowsInserted,&app,[&](const QModelIndex&,int,int) {++insertions;});
        QJsonArray rows{QJsonObject{{"id",1},{"name","Self"},{"damage",100},{"dps",100},{"share",40}},QJsonObject{{"id",2},{"name","Party"},{"damage",150},{"dps",150},{"share",60}}};
        model.update(rows);
        QElapsedTimer timing;timing.start();
        for(int i=0;i<10000;++i) model.update(rows);
        auto elapsed=timing.elapsed();
        QJsonArray reordered{rows[1],rows[0]};model.update(reordered);
        if(resets!=0||insertions!=2||model.rows[0].value("id").toInt()!=2) {app.exit(5);return;}
        fprintf(stderr,"Model check: 10000 stable updates in %lldms; no model resets.\n",elapsed);
        const auto nativePosition=window->position();const auto nativeSize=window->size();
        const auto pos=bridge.position();bridge.beginDrag();bridge.dragOverlay(pos.x()>12?-12:12,pos.y()>8?-8:8);
        if(bridge.position()==pos) {app.exit(4);return;}bridge.dragging=false;bridge.positionOverlay(pos);
        if(window->position()!=nativePosition||window->size()!=nativeSize||window->mask()!=QRegion(QRectF(bridge.panel->x(),bridge.panel->y(),bridge.panel->width(),bridge.panel->height()).toAlignedRect())) {app.exit(8);return;}
        if(!headless&&!bridge.gameScreen.isEmpty()&&window->screen()->name()!=bridge.gameScreen) {app.exit(9);return;}
        if(!headless) {
            auto originalScreen=window->screen();auto follow=bridge.autoDisplay;bridge.autoDisplay=false;
            for(auto target:QGuiApplication::screens()) {
                bridge.selectScreen(target);QEventLoop frames;QTimer::singleShot(80,&frames,&QEventLoop::quit);frames.exec();
                if(window->screen()!=target||window->isActive()) {app.exit(60);return;}
            }
            bridge.selectScreen(originalScreen);bridge.autoDisplay=follow;
        }
        // Check content sizing, uniform settings, and bounds while keeping the native surface still.
        bridge.publish.stop();
        bridge.bossState.clear();emit bridge.bossChanged();
        auto settle=[&] {QEventLoop frames;QTimer::singleShot(60,&frames,&QEventLoop::quit);window->requestUpdate();frames.exec();};
        bridge.actors.update({});settle();const auto emptyHeight=bridge.panel->height();
        QJsonArray partyRows;
        for(int i=0;i<4;++i) partyRows.append(QJsonObject{{"id",i+1},{"name",QString("Player %1").arg(i+1)},{"class",QStringList{"Templar","Sorcerer","Cleric","Assassin"}[i]},{"damage",400-i*80},{"dps",200-i*40},{"fill",100-i*20},{"share",25}});
        bridge.actors.update(partyRows);settle();const auto partyHeight=bridge.panel->height();
        auto bars=window->findChild<QQuickItem*>("playerBars");
        if(!bars||partyHeight<=emptyHeight||qAbs(bars->height()-(4*bridge.barHeight()+3*bridge.barSpacing()))>1) {fprintf(stderr,"Content sizing failed: empty=%f party=%f bars=%f\n",emptyHeight,partyHeight,bars?bars->height():-1);app.exit(10);return;}
        if(!QMetaObject::invokeMethod(window,"openSettings")) {app.exit(11);return;}
        settle();
        QQuickWindow *settingsWindow=nullptr;
        for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="settingsWindow") settingsWindow=qobject_cast<QQuickWindow*>(candidate);
        if(!settingsWindow||settingsWindow->transientParent()||!settingsWindow->isVisible()||(settingsWindow->flags()&Qt::WindowDoesNotAcceptFocus)||qAbs(bridge.panel->height()-partyHeight)>1||window->size()!=nativeSize) {app.exit(12);return;}
        auto moveSlider=[&](const char *name,int value) {
            auto slider=settingsWindow->findChild<QObject*>(name);
            return slider&&slider->setProperty("value",value)&&QMetaObject::invokeMethod(slider,"moved");
        };
        if(bridge.backgroundOpacity()!=75||bridge.overallOpacity()!=100) {app.exit(21);return;}
        if(!moveSlider("barWidthSlider",440)||!moveSlider("barHeightSlider",32)||!moveSlider("barSpacingSlider",10)) {app.exit(13);return;}
        settle();
        if(qAbs(bars->height()-(4*32+3*10))>1||qAbs(bridge.panel->width()-qMin(468,window->width()))>1) {app.exit(13);return;}
        auto styleSelector=settingsWindow->findChild<QObject*>("barStyleSelector");
        if(!styleSelector||bridge.barStyle()!=0) {app.exit(42);return;}
        auto stylePopup=styleSelector->property("popup").value<QObject*>();
        if(!stylePopup||!QMetaObject::invokeMethod(stylePopup,"open")) {app.exit(61);return;}
        settle();if(!stylePopup->property("visible").toBool()||(!headless&&window->isActive())) {app.exit(62);return;}
        QMetaObject::invokeMethod(stylePopup,"close");settle();
        for(int style=0;style<4;++style) {
            styleSelector->setProperty("currentIndex",style);
            if(!QMetaObject::invokeMethod(styleSelector,"activated",Q_ARG(int,style))||bridge.barStyle()!=style) {app.exit(43);return;}
            settle();Bridge restored;if(restored.barStyle()!=style) {app.exit(44);return;}
            const auto path=qEnvironmentVariable("AIONDPS_TEST_SCREENSHOT");
            if(!path.isEmpty()) window->grabWindow().save(path+QString("-style-%1.png").arg(style));
        }
        bridge.setBarStyle(0);
        QQuickItem *compact=nullptr;
        QList<QQuickItem*> pending{bars};
        while(!pending.isEmpty()) {auto item=pending.takeFirst();if(item->objectName()=="compactShare") {compact=item;break;}pending.append(item->childItems());}
        if(!compact||!compact->isVisible()) {app.exit(45);return;}
        auto hpBar=window->findChild<QQuickItem*>("targetHealthBar");
        auto etaBar=window->findChild<QQuickItem*>("targetKillTimeBar");
        auto etaLabel=window->findChild<QObject*>("targetKillTimeLabel");
        auto wipeLabel=window->findChild<QObject*>("targetWipeCountdownLabel");
        auto elapsedLabel=window->findChild<QObject*>("targetElapsedTimeLabel");
        const auto savedBoss=bridge.bossState;bridge.bossState.clear();emit bridge.bossChanged();settle();
        if(!hpBar||hpBar->isVisible()||!etaBar||etaBar->isVisible()||!etaLabel||!wipeLabel||!elapsedLabel) {app.exit(46);return;}
        QVariant greenColor,yellowLow,yellowHigh,redColor;
        QMetaObject::invokeMethod(window,"etaColor",Q_RETURN_ARG(QVariant,greenColor),Q_ARG(QVariant,QVariant(239.0)),Q_ARG(QVariant,QVariant(true)));
        QMetaObject::invokeMethod(window,"etaColor",Q_RETURN_ARG(QVariant,yellowLow),Q_ARG(QVariant,QVariant(240.0)),Q_ARG(QVariant,QVariant(true)));
        QMetaObject::invokeMethod(window,"etaColor",Q_RETURN_ARG(QVariant,yellowHigh),Q_ARG(QVariant,QVariant(300.0)),Q_ARG(QVariant,QVariant(true)));
        QMetaObject::invokeMethod(window,"etaColor",Q_RETURN_ARG(QVariant,redColor),Q_ARG(QVariant,QVariant(301.0)),Q_ARG(QVariant,QVariant(true)));
        if(greenColor.toString()!="#54cf82"||yellowLow.toString()!="#f1c778"||yellowHigh.toString()!="#f1c778"||redColor.toString()!="#ee6464") {app.exit(46);return;}
        const auto savedSpan=bridge.span;
        bridge.span=240.0;emit bridge.durationChanged();
        bridge.bossState=QVariantMap{{"entity",900100},{"name","Fediv Wraith"},{"hp",900000},{"max",8000000},{"percent",11.25},{"eta",90.0},{"totalEta",330.0},{"timeLeft",60.0},{"etaKnown",true},{"known",true},{"maxKnown",true}};
        emit bridge.bossChanged();settle();
        if(etaBar->property("statusColor").value<QColor>().name().toLower()!="#ee6464"||!wipeLabel->property("text").toString().contains("1:00")||!elapsedLabel->property("text").toString().contains("4:00")||!etaLabel->property("text").toString().contains("1:30")) {app.exit(46);return;}
        bridge.span=savedSpan;emit bridge.durationChanged();
        bridge.bossState=savedBoss;emit bridge.bossChanged();settle();
        if(!moveSlider("backgroundOpacitySlider",20)) {app.exit(22);return;}
        settle();
        if(qAbs(bridge.panel->property("color").value<QColor>().alphaF()-0.2)>0.01||bridge.panel->opacity()!=1||bars->opacity()!=1) {app.exit(22);return;}
        if(!moveSlider("overallOpacitySlider",45)) {app.exit(23);return;}
        settle();
        if(qAbs(bridge.panel->opacity()-0.45)>0.01||settingsWindow->opacity()!=1||!settingsWindow->isVisible()||qAbs(bridge.panel->property("color").value<QColor>().alphaF()-0.2)>0.01) {app.exit(23);return;}
        if(!moveSlider("overallOpacitySlider",0)) {app.exit(24);return;}
        settle();
        if(window->mask()!=QRegion(QRect(-1,-1,1,1))||!settingsWindow->isVisible()) {app.exit(24);return;}
        settingsWindow->close();
        QMetaObject::invokeMethod(window,"openSettings");settle();
        if(!settingsWindow->isVisible()||QGuiApplication::allWindows().size()!=2) {app.exit(25);return;}
        // Values persisted by actual slider signals can be loaded by a fresh bridge.
        {Bridge restored;if(restored.barWidth()!=440||restored.barHeight()!=32||restored.barSpacing()!=10||restored.backgroundOpacity()!=20||restored.overallOpacity()!=0) {app.exit(26);return;}}
        bridge.setScoreDisplay(2);bridge.setShowTotalDamage(false);bridge.setRoundedDps(true);settle();
        {Bridge restored;if(restored.scoreDisplay()!=2||restored.showTotalDamage()||!restored.roundedDps()) {fprintf(stderr,"Bar information preferences were not persisted\n");app.exit(47);return;}}
        QVariant formatted;QMetaObject::invokeMethod(window,"dpsText",Q_RETURN_ARG(QVariant,formatted),Q_ARG(QVariant,QVariant(12345.0)));
        if(!formatted.toString().endsWith("K")) {fprintf(stderr,"Rounded DPS formatting failed\n");app.exit(48);return;}
        auto minimalistToggle=settingsWindow->findChild<QObject*>("minimalistModeToggle");
        auto minimalistCompactToggle=settingsWindow->findChild<QQuickItem*>("minimalistCompactDamageToggle");
        if(!minimalistToggle||!minimalistCompactToggle||minimalistCompactToggle->isVisible()||bridge.minimalistMode()||!bridge.minimalistCompactDamage()) {app.exit(90);return;}
        minimalistToggle->setProperty("checked",true);
        if(!QMetaObject::invokeMethod(minimalistToggle,"toggled")||!bridge.minimalistMode()||!minimalistCompactToggle->isVisible()) {app.exit(90);return;}
        settle();
        {Bridge restored;if(!restored.minimalistMode()||!restored.minimalistCompactDamage()) {app.exit(90);return;}}
        QVariant compactDmg,fullDmg,compactScore,fullScore,compactHp,fullHp;
        QMetaObject::invokeMethod(window,"minimalistDamageText",Q_RETURN_ARG(QVariant,compactDmg),Q_ARG(QVariant,QVariant(1523.0)));
        QMetaObject::invokeMethod(window,"scoreValueText",Q_RETURN_ARG(QVariant,compactScore),Q_ARG(QVariant,QVariant(102301.0)));
        QMetaObject::invokeMethod(window,"bossHpValueText",Q_RETURN_ARG(QVariant,compactHp),Q_ARG(QVariant,QVariant(8000000.0)));
        if(!compactDmg.toString().endsWith("K")||!compactScore.toString().endsWith("K")||!compactHp.toString().endsWith("M")) {app.exit(92);return;}
        minimalistCompactToggle->setProperty("checked",false);
        if(!QMetaObject::invokeMethod(minimalistCompactToggle,"toggled")||bridge.minimalistCompactDamage()) {app.exit(92);return;}
        {Bridge restored;if(restored.minimalistCompactDamage()) {app.exit(92);return;}}
        QMetaObject::invokeMethod(window,"minimalistDamageText",Q_RETURN_ARG(QVariant,fullDmg),Q_ARG(QVariant,QVariant(6741514.0)));
        QMetaObject::invokeMethod(window,"scoreValueText",Q_RETURN_ARG(QVariant,fullScore),Q_ARG(QVariant,QVariant(102301.0)));
        QMetaObject::invokeMethod(window,"bossHpValueText",Q_RETURN_ARG(QVariant,fullHp),Q_ARG(QVariant,QVariant(8000000.0)));
        if(fullDmg.toString().endsWith("M")||fullDmg.toString().endsWith("K")||fullScore.toString().endsWith("K")||fullHp.toString().endsWith("M")) {app.exit(92);return;}
        QQuickItem *minStats=nullptr,*dmgFill=nullptr,*nameLbl=nullptr;
        QList<QQuickItem*> searchQueue{bars};
        while(!searchQueue.isEmpty()) {
            auto item=searchQueue.takeFirst();
            if(item->objectName()=="minimalistStats"&&!minStats) minStats=item;
            if(item->objectName()=="damageFill"&&!dmgFill) dmgFill=item;
            if(item->objectName()=="actorNameLabel"&&!nameLbl) nameLbl=item;
            searchQueue.append(item->childItems());
        }
        if(!minStats||!minStats->isVisible()||!dmgFill||dmgFill->isVisible()||!nameLbl||nameLbl->property("color").value<QColor>().name().toLower()!="#5f8cff"||qAbs(bars->height()-(4*22+3*10))>1) {app.exit(91);return;}
        bridge.resetAppearance();settle();
        if(bridge.minimalistMode()||!bridge.minimalistCompactDamage()||bridge.scoreDisplay()!=1||!bridge.showTotalDamage()||bridge.roundedDps()) {app.exit(49);return;}
        if(bridge.backgroundOpacity()!=75||bridge.overallOpacity()!=100||qAbs(bridge.panel->height()-partyHeight)>1||window->mask()!=QRegion(QRectF(bridge.panel->x(),bridge.panel->y(),bridge.panel->width(),bridge.panel->height()).toAlignedRect())) {app.exit(27);return;}
        const auto previousCharacterName=bridge.characterName();
        bridge.setCharacterName("RememberedPlayer");
        {Bridge restored;if(restored.characterName()!="RememberedPlayer"||restored.character()!="RememberedPlayer") {app.exit(88);return;}}
        bridge.setTestMode(true);settle();
        if(bridge.characterName()!="RememberedPlayer") {app.exit(89);return;}
        bridge.setTestMode(false);bridge.setCharacterName(previousCharacterName);settle();
        fprintf(stderr,"Remembered character setting, startup fallback and preview isolation checks passed.\n");
        const auto growthPosition=bridge.position();
        bridge.setGrowUp(true);bridge.positionOverlay({growthPosition.x(),window->height()-qRound(bridge.panel->height())-20});settle();
        const auto fixedBottom=bridge.panel->y()+bridge.panel->height();
        bridge.actors.update({});settle();
        if(qAbs(bridge.panel->y()+bridge.panel->height()-fixedBottom)>1) {fprintf(stderr,"Upward shrink moved bottom edge\n");app.exit(80);return;}
        bridge.actors.update(partyRows);settle();
        if(qAbs(bridge.panel->y()+bridge.panel->height()-fixedBottom)>1) {fprintf(stderr,"Upward growth moved bottom edge\n");app.exit(81);return;}
        {Bridge restored;if(!restored.growUp()) {app.exit(82);return;}}
        bridge.setGrowUp(false);bridge.positionOverlay(growthPosition);const auto fixedTop=bridge.panel->y();
        bridge.actors.update({});settle();bridge.actors.update(partyRows);settle();
        if(qAbs(bridge.panel->y()-fixedTop)>1) {fprintf(stderr,"Downward growth moved top edge\n");app.exit(83);return;}
        fprintf(stderr,"Up/down bar growth, fixed edges and saved preference checks passed.\n");
        auto screenshot=qEnvironmentVariable("AIONDPS_TEST_SCREENSHOT");
        if(!screenshot.isEmpty()) {window->grabWindow().save(screenshot);settingsWindow->grabWindow().save(screenshot+"-settings.png");}
        // Preview uses the same live delegates and details models, never real history.
        const auto realState=bridge.currentState;const auto realHistory=bridge.history;
        // Idle clearing retains the archived fight, but the next session replaces it.
        if(!realHistory.isEmpty()) {
            const auto archived=realHistory.first().toMap().value("snapshot").toMap();
            auto idle=realState;idle["session"]=archived.value("session");idle["actors"]=QVariantList{};idle["active"]=false;idle["character"]="Current character";
            bridge.currentState=idle;bridge.refreshDisplay();
            if(bridge.displayPlayers().isEmpty()||bridge.displayState!=archived||bridge.character()!="Current character") {app.exit(39);return;}
            idle["session"]=archived.value("session").toULongLong()+1;idle["active"]=true;idle["actors"]=realState.value("actors");
            bridge.currentState=idle;bridge.refreshDisplay();
            if(bridge.displayState!=idle) {app.exit(40);return;}
            bridge.currentState=realState;bridge.refreshDisplay();
        }
        auto versionLabel=window->findChild<QObject*>("appVersionLabel");
        if(!versionLabel||versionLabel->property("text").toString()!=QString("v")+SLOPMETER_VERSION) {app.exit(59);return;}
        // Alpha ordering is numeric; published drafts and malformed tags cannot advertise updates.
        const QList<QPair<QString,QString>> newerPairs{{"0.1.0-alpha.10","0.1.0-alpha.2"},{"0.1.0","0.1.0-rc.1"},{"1.0.0-alpha","0.9.9"},{"1.0.0-alpha.beta","1.0.0-alpha.1"}};
        for(const auto &pair:newerPairs) {
            ReleaseVersion newer(pair.first),older(pair.second);
            if(!newer.valid||!older.valid||newer.compare(older)<=0||older.compare(newer)>=0) {app.exit(65);return;}
        }
        if(ReleaseVersion("v0.1.0-alpha.3+build.2").compare(ReleaseVersion("0.1.0-alpha.3"))!=0||ReleaseVersion("0.1.0-alpha.03").valid||ReleaseVersion("broken").valid) {app.exit(66);return;}
        const QByteArray releases=R"([{"tag_name":"v0.1.0-alpha.10","published_at":"2026-10-04","prerelease":true},{"tag_name":"v99.0.0","published_at":"2026-10-04","draft":true},{"tag_name":"bad","published_at":"2026-10-04"},{"tag_name":"v98.0.0"}])";
        ReleaseCheck alpha("0.1.0-alpha.3"),stable("0.1.0");alpha.applyReleases(releases);stable.applyReleases(releases);
        if(!alpha.available()||alpha.version()!="0.1.0-alpha.10"||stable.available()) {app.exit(67);return;}
        alpha.applyReleases("{error}");if(!alpha.available()) {app.exit(68);return;}
        alpha.applyReleases(R"([{"tag_name":"v0.1.0-alpha.3","published_at":"2026-10-04"}])");
        if(alpha.available()) {app.exit(69);return;}
        auto updateLink=window->findChild<QQuickItem*>("updateAvailableLink");
        if(!updateLink||updateLink->isVisible()) {app.exit(70);return;}
        updates.applyReleases(R"([{"tag_name":"v999999999999.0.0","published_at":"2026-10-04"}])");settle();
        if(!updateLink->isVisible()||updateLink->property("text").toString()!="Update available"||updates.url()!="https://github.com/Netskill89/slopmeter/releases"||updateLink->mapToItem(bridge.panel,{updateLink->width(),0}).x()>bridge.panel->width()) {app.exit(71);return;}
        updates.applyReleases("[]");settle();if(updateLink->isVisible()) {app.exit(72);return;}
        fprintf(stderr,"Release comparison, alpha updates, malformed/offline replies and footer link checks passed.\n");
        const auto stateBeforeScene=bridge.currentState;
        const auto selectionBeforeScene=bridge.selectedSession;bridge.selectedSession=0;
        bridge.currentState["scene"]=QVariantMap{{"id",600021},{"name","Fire Temple"},{"instance",42}};bridge.refreshDisplay();settle();
        auto combatLabel=window->findChild<QObject*>("combatStatus");
        if(bridge.sceneName()!="Fire Temple"||!combatLabel||!combatLabel->property("text").toString().contains("Fire Temple")) {app.exit(75);return;}
        bridge.currentState["pvp"]=true;
        bridge.currentState["pvpSummary"]=QVariantMap{{"opponentId",99},{"opponentName","ShadowBlade"},{"opponentClass","Assassin"},{"damageDealt",25000},{"damageTaken",18000},{"dealtDps",2500},{"takenDps",1800},{"dealtShare",58.1},{"outcome","Victory"}};
        bridge.currentState["enemies"]=QVariantList{QVariantMap{{"id",99},{"name","ShadowBlade"},{"class","Assassin"},{"enemy",true},{"damage",18000},{"damageTaken",25000},{"dps",1800},{"criticalRate",45.0},{"criticalKnown",true},{"maxHit",4200},{"share",100},{"fill",100},{"skills",QVariantList{QVariantMap{{"id",12010000},{"name","Ambush"},{"damage",18000},{"share",100},{"fightShare",100},{"hits",5},{"uses",1},{"usesKnown",true},{"critical",2},{"criticalRate",40},{"criticalKnown",true},{"min",3000},{"max",4200},{"average",3600},{"hitsPerSecond",0.5}}}}}};
        bridge.refreshDisplay();settle();
        auto pvpHeader=window->findChild<QQuickItem*>("pvpDuelHeader");
        auto enemySec=window->findChild<QQuickItem*>("enemySection");
        auto pvpTitle=window->findChild<QObject*>("pvpHeaderTitle");
        if(!bridge.isPvP()||!pvpHeader||!pvpHeader->isVisible()||!enemySec||!enemySec->isVisible()||bridge.enemies.rowCount()!=1||!pvpTitle||!pvpTitle->property("text").toString().contains("ShadowBlade")||!pvpTitle->property("text").toString().contains("VICTORY")) {app.exit(95);return;}
        bridge.selectPlayer(99);
        if(!bridge.detailPlayer().value("enemy").toBool()||bridge.detailPlayer().value("name").toString()!="ShadowBlade"||bridge.skills.rowCount()!=1) {app.exit(96);return;}
        // Radar 2D map, subcategories & elevation callout checks
        auto radarSec=window->findChild<QQuickItem*>("radarSection");
        auto radarBtn=window->findChild<QQuickItem*>("radarToggleButton");
        auto radarMap=window->findChild<QQuickItem*>("radarMapCanvas");
        auto radarCenter=window->findChild<QQuickItem*>("radarPlayerCenter");
        auto radarSelName=window->findChild<QObject*>("radarSelectedName");
        auto radarSelVert=window->findChild<QObject*>("radarSelectedVert");
        if(!radarSec||!radarBtn||!radarMap||!radarCenter||!radarSelName||!radarSelVert||radarSec->isVisible()||bridge.radarEnabled()) {app.exit(97);return;}
        bridge.currentState["radar"]=QVariantList{
            QVariantMap{{"id",501},{"npc",1200039},{"name","Hidden Cube"},{"category","chest"},{"distance",18.0},{"zDelta",5.0},{"angle",0.0},{"relX",0.0},{"relY",18.0},{"horizDir","Frente"},{"vertDir","Arriba"},{"hasDir",true}},
            QVariantMap{{"id",502},{"npc",1080001},{"name","Odyle"},{"category","odyle"},{"distance",24.0},{"zDelta",-4.0},{"angle",90.0},{"relX",24.0},{"relY",0.0},{"horizDir","Der"},{"vertDir","Abajo"},{"hasDir",true}},
            QVariantMap{{"id",503},{"npc",1080501},{"name","Sapphire Gemstone"},{"category","gem"},{"distance",28.0},{"zDelta",3.0},{"angle",-45.0},{"relX",-19.8},{"relY",19.8},{"horizDir","Frente"},{"vertDir","Arriba"},{"hasDir",true}},
            QVariantMap{{"id",504},{"npc",1080101},{"name","Orichalcum Ore"},{"category","ore"},{"distance",30.0},{"zDelta",0.0},{"angle",-90.0},{"relX",-30.0},{"relY",0.0},{"horizDir","Izq"},{"vertDir","Nivel"},{"hasDir",true}},
            QVariantMap{{"id",505},{"npc",1090601},{"name","Azpha"},{"category","herb"},{"distance",35.0},{"zDelta",-2.5},{"angle",180.0},{"relX",0.0},{"relY",-35.0},{"horizDir","Atrás"},{"vertDir","Abajo"},{"hasDir",true}},
            QVariantMap{{"id",506},{"npc",1090201},{"name","Asvata Log"},{"category","wood"},{"distance",38.0},{"zDelta",0.0},{"angle",45.0},{"relX",26.8},{"relY",26.8},{"horizDir","Frente"},{"vertDir","Nivel"},{"hasDir",true}},
            QVariantMap{{"id",507},{"npc",2999999},{"name","Mystery Box"},{"category","object"},{"distance",40.0},{"zDelta",0.0},{"angle",180.0},{"relX",0.0},{"relY",-40.0},{"horizDir","Atrás"},{"vertDir","Nivel"},{"hasDir",true}}
        };
        bridge.refreshDisplay();
        if(!QMetaObject::invokeMethod(radarBtn,"clicked")||!bridge.radarEnabled()) {app.exit(97);return;}
        settle();
        if(!radarSec->isVisible()||!radarMap->isVisible()||bridge.displayRadar().size()!=6||!radarSelName->property("text").toString().contains("Hidden Cube")||!radarSelVert->property("text").toString().contains("ARRIBA")) {app.exit(97);return;}
        radarSec->setProperty("selectedRadarId",502);settle();
        if(!radarSelName->property("text").toString().contains("Odyle")||!radarSelVert->property("text").toString().contains("ABAJO")) {app.exit(97);return;}
        radarSec->setProperty("selectedRadarId",0);
        bridge.setRadarItemFilter("Sapphire Gemstone");settle();
        if(bridge.displayRadar().size()!=1||!radarSelName->property("text").toString().contains("Sapphire Gemstone")) {app.exit(98);return;}
        bridge.setRadarItemFilter(QString());
        bridge.setRadarChests(false);bridge.setRadarGems(false);settle();
        if(bridge.displayRadar().size()!=4) {app.exit(98);return;}
        bridge.setRadarOther(true);settle();
        if(bridge.displayRadar().size()!=5) {app.exit(98);return;}
        {Bridge restored;if(!restored.radarEnabled()||restored.radarChests()||restored.radarGems()||!restored.radarOther()) {app.exit(98);return;}}
        bridge.setRadarChests(true);bridge.setRadarGems(true);bridge.setRadarOther(false);bridge.setRadarEnabled(false);settle();
        bridge.currentState=stateBeforeScene;bridge.selectedSession=selectionBeforeScene;bridge.refreshDisplay();settle();
        fprintf(stderr,"Scene metadata, PvP duel mode, enemy split bars, rival detail and 6-way radar checks passed.\n");
        auto resetButton=window->findChild<QQuickItem*>("resetDamageButton");
        auto settingsButton=window->findChild<QQuickItem*>("settingsButton");
 auto minimizeButton=window->findChild<QQuickItem*>("minimizeButton");
 if(!resetButton||!settingsButton||!minimizeButton||resetButton->x()<=settingsButton->x()||resetButton->x()>=minimizeButton->x()) {app.exit(76);return;}
        if(bridge.process.state()==QProcess::NotRunning) {
            const auto stateBeforeReset=bridge.currentState;const auto historyBeforeReset=bridge.history;
            if(!QMetaObject::invokeMethod(resetButton,"clicked")) {app.exit(77);return;}
            settle();
            if(!bridge.displayPlayers().isEmpty()||bridge.duration()!=0||bridge.history!=historyBeforeReset) {app.exit(78);return;}
            bridge.currentState=stateBeforeReset;bridge.refreshDisplay();settle();
        }

        // Portable desktop registration stays hidden and preserves installed launchers.
        QTemporaryDir desktopData;if(!desktopData.isValid()) {app.exit(73);return;}
        const auto oldDataHome=qgetenv("XDG_DATA_HOME"),oldDataDirs=qgetenv("XDG_DATA_DIRS");
        qputenv("XDG_DATA_HOME",desktopData.path().toUtf8());qputenv("XDG_DATA_DIRS",desktopData.path().toUtf8());
        const bool registered=registerDesktopIdentity();
        QFile desktopEntry(desktopData.path()+"/applications/slopmeter.desktop");
        bool desktopValid=registered&&desktopEntry.open(QIODevice::ReadOnly);
        const auto entry=desktopEntry.readAll();desktopEntry.close();
        desktopValid=desktopValid&&entry.contains("NoDisplay=true\n")&&entry.contains("Exec=")&&QFile::exists(desktopData.path()+"/icons/hicolor/scalable/apps/slopmeter.svg");
        if(desktopEntry.open(QIODevice::WriteOnly)) {desktopEntry.write("existing launcher\n");desktopEntry.close();} else desktopValid=false;
        desktopValid=desktopValid&&registerDesktopIdentity()&&desktopEntry.open(QIODevice::ReadOnly)&&desktopEntry.readAll()=="existing launcher\n";desktopEntry.close();
        if(oldDataHome.isNull()) qunsetenv("XDG_DATA_HOME");else qputenv("XDG_DATA_HOME",oldDataHome);
        if(oldDataDirs.isNull()) qunsetenv("XDG_DATA_DIRS");else qputenv("XDG_DATA_DIRS",oldDataDirs);
        if(!desktopValid) {app.exit(74);return;}
        fprintf(stderr,"Portable desktop identity and installed-launcher preservation checks passed.\n");
        auto captureDot=window->findChild<QQuickItem*>("captureIndicator");
        auto displayText=window->findChild<QObject*>("displayLabel");
        if(!captureDot||!displayText||!displayText->property("text").toString().startsWith("Display - ")||bridge.captureActive()) {app.exit(41);return;}
        int peakPreviewRows=bridge.actors.rowCount();QObject::connect(&bridge.actors,&QAbstractItemModel::rowsInserted,&app,[&](const QModelIndex&,int,int) {peakPreviewRows=qMax(peakPreviewRows,bridge.actors.rowCount());});
        auto toggle=[&](const char *name,bool enabled) {
            auto control=settingsWindow->findChild<QObject*>(name);
            return control&&control->setProperty("checked",enabled)&&QMetaObject::invokeMethod(control,"toggled");
        };
        if(bridge.pollInterval()!=200||!bridge.showBorder()) {app.exit(28);return;}
        if(!toggle("testModeToggle",true)) {app.exit(29);return;}
        settle();
        if(!bridge.testMode()||bridge.actors.rowCount()!=5||peakPreviewRows>5||bridge.character()!="Test party"||bridge.skills.rowCount()!=8||!bridge.boss().value("etaKnown").toBool()||bridge.boss().value("eta").toDouble()<=0) {app.exit(29);return;}
        double damageBeforeReset=0;for(auto value:bridge.previewDamage) damageBeforeReset+=value;
        if(!QMetaObject::invokeMethod(resetButton,"clicked")) {app.exit(79);return;}
        settle();double damageAfterReset=0;for(auto value:bridge.previewDamage) damageAfterReset+=value;
        if(bridge.actors.rowCount()!=5||bridge.duration()>1||damageAfterReset>=damageBeforeReset/10) {app.exit(80);return;}
        fprintf(stderr,"Manual reset button, history preservation and preview reset checks passed.\n");

        auto previewHealth=window->findChild<QQuickItem*>("targetHealthBar");
        if(!previewHealth||!previewHealth->isVisible()||!etaBar||!etaBar->isVisible()||!bridge.boss().value("maxKnown").toBool()||bridge.boss().value("percent").toDouble()<=0||bridge.boss().value("percent").toDouble()>=100) {app.exit(49);return;}
        if(!toggle("borderToggle",false)||QQmlProperty(bridge.panel,"border.width").read().toInt()!=0) {app.exit(30);return;}
        if(!moveSlider("barWidthSlider",480)||!moveSlider("barHeightSlider",40)||!moveSlider("barSpacingSlider",8)) {app.exit(31);return;}
        settle();
        if(qAbs(bars->height()-(5*40+4*8))>1||qAbs(bridge.panel->width()-qMin(508,window->width()))>1) {app.exit(31);return;}
        if(!moveSlider("pollIntervalSlider",50)||bridge.publish.interval()!=50) {app.exit(32);return;}
        settle();
        auto warning=settingsWindow->findChild<QQuickItem*>("cpuWarning");
        if(!warning||!warning->isVisible()) {app.exit(32);return;}
        const auto damageBefore=bridge.displayPlayers().first().toMap().value("damage").toLongLong();settle();
        if(bridge.displayPlayers().first().toMap().value("damage").toLongLong()<=damageBefore||bridge.currentState!=realState||bridge.history!=realHistory) {app.exit(33);return;}
        if(!moveSlider("pollIntervalSlider",1000)||bridge.publish.interval()!=1000) {app.exit(34);return;}
        if(warning->isVisible()) {app.exit(34);return;}
        {Bridge restored;if(restored.pollInterval()!=1000||restored.showBorder()||restored.testMode()) {app.exit(35);return;}}
        for(int rate=50;rate<=1000;rate+=50) {bridge.setPollInterval(rate);if(bridge.pollInterval()!=rate) {app.exit(36);return;}}
        bridge.setPollInterval(200);
        {
            auto selector=settingsWindow->findChild<QObject*>("captureInterfaceSelector");
            if(!selector||selector->property("enabled").toBool()) {app.exit(50);return;}
            bridge.interfaceRows={QVariantMap{{"name",""},{"label","Automatic"}},QVariantMap{{"name","any"},{"label","All interfaces"}}};emit bridge.interfacesChanged();settle();
            if(selector->property("currentIndex").toInt()!=0) {app.exit(56);return;}
            bridge.interfaceSetting="any";emit bridge.interfacesChanged();settle();
            if(selector->property("currentIndex").toInt()!=1) {app.exit(57);return;}
            bridge.interfaceSetting="";emit bridge.interfacesChanged();
            Bridge capture;
            capture.backendPath="/bin/sh";
            capture.captureArguments={"-c","trap 'exit 0' TERM; while :; do sleep 0.05; done","capture-test"};
            capture.setCaptureInterface("");capture.startCapture();
            if(!capture.process.waitForStarted(1000)||capture.process.arguments().contains("-interface")) {app.exit(51);return;}
            const auto initialPID=capture.process.processId();capture.applyCaptureInterface();
            if(capture.restartingCapture||capture.process.processId()!=initialPID) {app.exit(58);return;}
            capture.setCaptureInterface("any");
            {Bridge restored;if(restored.captureInterface()!="any") {app.exit(52);return;}}
            if(capture.process.arguments().contains("-interface")) {app.exit(53);return;}
            QEventLoop restarted;
            QObject::connect(&capture.process,&QProcess::started,&restarted,&QEventLoop::quit);
            QTimer::singleShot(3500,&restarted,&QEventLoop::quit);
            capture.applyCaptureInterface();restarted.exec();
            if(capture.restartingCapture||capture.process.state()!=QProcess::Running||capture.process.arguments().last()!="any") {app.exit(54);return;}
            capture.replayMode=true;capture.setCaptureInterface("lo");capture.applyCaptureInterface();
            if(capture.captureInterface()!="any"||capture.restartingCapture) {app.exit(55);return;}
            capture.replayMode=false;capture.setCaptureInterface("");
            fprintf(stderr,"Network interface persistence, explicit apply, restart and replay isolation checks passed.\n");
        }
        if(!screenshot.isEmpty()) {
            window->grabWindow().save(screenshot+"-preview.png");settingsWindow->grabWindow().save(screenshot+"-preview-settings.png");
            auto capture=bridge.panel->grabToImage();QEventLoop rendered;
            QObject::connect(capture.get(),&QQuickItemGrabResult::ready,&rendered,&QEventLoop::quit);QTimer::singleShot(1000,&rendered,&QEventLoop::quit);rendered.exec();
            capture->image().save(screenshot+"-meter.png");
        }
        if(!toggle("testModeToggle",false)) {app.exit(37);return;}
        if(bridge.currentState!=realState||bridge.history!=realHistory||bridge.displayState!=realState||bridge.publish.isActive()) {app.exit(37);return;}
        bridge.resetAppearance();settle();
        if(!bridge.showBorder()||QQmlProperty(bridge.panel,"border.width").read().toInt()!=1) {app.exit(38);return;}
        settingsWindow->close();
        // Historical selection and live detail models use the same data, with no resets.
        if(!bridge.history.isEmpty()) {
            const auto archived=bridge.history.first().toMap().value("snapshot").toMap();
            bridge.selectFight(archived.value("session").toULongLong());
            if(bridge.detailFight().value("session")!=archived.value("session")||bridge.displayPlayers().isEmpty()) {app.exit(14);return;}
            bridge.selectFight(0);
        }
        if(!bridge.displayPlayers().isEmpty()) {
            const auto actor=bridge.displayPlayers().first().toMap().value("id").toUInt();
            bridge.selectPlayer(actor);
            if(bridge.skills.rowCount()==0) {app.exit(15);return;}
            int skillResets=0;QObject::connect(&bridge.skills,&QAbstractItemModel::modelReset,&app,[&] {++skillResets;});
            for(int i=0;i<1000;++i) bridge.updateDetail();
            if(skillResets) {app.exit(16);return;}
            if(!QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)))) {app.exit(17);return;}
            settle();
            QQuickWindow *detailWindow=nullptr;
            for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="fightDetailsWindow") detailWindow=qobject_cast<QQuickWindow*>(candidate);
            if(!detailWindow||detailWindow->transientParent()||!detailWindow->isVisible()||(detailWindow->flags()&Qt::WindowDoesNotAcceptFocus)||window->size()!=nativeSize) {app.exit(18);return;}
            auto tabs=detailWindow->findChild<QObject*>("analysisTabs");
            auto timelineView=detailWindow->findChild<QQuickItem*>("skillUsageTimeline");
            if(!tabs||!timelineView) {app.exit(84);return;}
            tabs->setProperty("currentIndex",1);settle();
            if(!timelineView->isVisible()) {app.exit(85);return;}
            bridge.setTestMode(true);settle();
            if(bridge.detailPlayer().value("timeline").toList().isEmpty()) {app.exit(86);return;}
            // Force a rendered frame of the populated timeline, catching QML/Canvas errors.
            if(detailWindow->grabWindow().isNull()) {app.exit(87);return;}
            bridge.setTestMode(false);settle();tabs->setProperty("currentIndex",0);settle();
            fprintf(stderr,"Skill timeline tab, preview events and rendering checks passed.\n");
            if(!screenshot.isEmpty()) detailWindow->grabWindow().save(screenshot+"-details.png");
            detailWindow->close();
            if(detailWindow->isVisible()||!window->isVisible()) {app.exit(19);return;}
            QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)));settle();
            if(!detailWindow->isVisible()||QGuiApplication::allWindows().size()!=3) {app.exit(20);return;}
            detailWindow->close();
        }
        fprintf(stderr,"Preview isolation, live layout, polling, border, opacity and secondary-window lifecycle checks passed on %s.\n",qPrintable(window->screen()->name()));app.quit();
    });
    return app.exec();
}
#include "main.moc"
