# Suite de pruebas de tiempo (P5-03)

El tiempo es lo único que este juego no puede hacer mal (GDD §6.1 y §8): el servidor es la única fuente de verdad, el progreso sale de
marcas de tiempo y nunca de contadores que se incrementan, y la hora del equipo del jugador no cuenta. Este documento enlaza cada regla
con la prueba que la demuestra. **Se ejecuta con `make test-time`** (`tools/time_suite.py`): comprueba que cada prueba nombrada existe y la
ejecuta; con `--e2e` ejecuta también los scripts de navegador.

## Criterios de aceptación del GDD §8

| # | Criterio | Pruebas (Go) | Navegador (E2E) |
| :-- | :--- | :--- | :--- |
| 1 | Cerrar el navegador a mitad de un Pomodoro y volver: tiempo exacto (±1 s), sin perder nada | `TestCloseBrowserAndReturnIsExact` · `TestCompletionTimeIsExactAfterLongAbsence` · `TestTheServerRestartingMidPomodoroLosesNothing` · `TestAPauseOfThirtyDaysChangesNothing` | `p1_04_timer.py` · `p5_03_time.py` |
| 2 | Dos dispositivos: el segundo ve el Pomodoro y no puede iniciar otro | `TestSecondDeviceCannotStartAnother` · `TestConcurrentPlantOnlyOneWins` · `TestOnlyOneActivePomodoroPerPlayer` · `TestTwoDevicesSeeTheSameRest` | `p5_03_time.py` |
| 3 | Cambiar la hora del sistema no altera ningún cálculo | `TestClockGoingBackwards` · `TestClockGoingBackwardsNeitherProducesNorDoubleCounts` · `TestTheServerClockJumpingForwardAndBackMidPomodoro` · `TestExtremeClocksNeitherPanicNorOverflow` · `TestTheServersTimeZoneNeverMatters` | `p5_03_time.py` (hora del equipo +10 h, −3 días, +1 año) · `p1_08_qa.py` |
| 4 | Ejemplo de §6.2 con los dos topes de Silo (212,7 y 127,0 🪙) | `TestGDDExampleWithTheTwoSiloCaps` · `TestGDDExampleRates` · `TestGDDExampleSiloCaps` | — |
| 5 | Ausencia de 90 días: como máximo lo que permite el Silo; nada produce más allá de su vida | `TestNinetyDaysAwayIsBounded` · `TestNinetyDaysAwayWithAFullFarm` · `TestNinetyDaysAwayIsBoundedBySiloAndLifespan` · `TestMaturesAndWiltsInsideTheSameAbsence` | — |
| 6 | Restaurar una copia en una instalación limpia deja el juego jugable | `TestBackupIsRestorableAndPrunes` · `TestUpgradingAnOldDatabaseKeepsEveryRow` | `tools/e2e/qa_restore.sh` (Docker limpio) |
| 7 | `sim.py` = hoja de cálculo (Normal 1.798,7 🪙/día) | `TestSteadyStateIncomeMatchesTheGDDFormula` | — |

## Reglas de tiempo que van más allá del §8

| Regla | Pruebas |
| :--- | :--- |
| **Liquidación perezosa (sin cron):** consultar el estado, a cualquier ritmo, no cambia dónde acaba la partida | `TestSettlingOftenEqualsSettlingOnce` · `TestSettlingOftenGivesTheSameResultAsOnce` · `TestPollingNeverChangesWhereTheGameEndsUp` (5 partidas aleatorias × 2 ritmos de consulta: 💧 y plantas idénticas, 🪙 a lo sumo una milésima de diferencia por redondeo) · `TestAskingForTheStateManyTimesAtOneInstantChangesNothing` |
| Nada se cuenta dos veces ni se pierde con peticiones simultáneas | `TestConcurrentRequestsDoNotDoubleCount` · `TestConcurrentPurchasesNeverSpendCoinsTwice` |
| Las pausas congelan el tiempo y no pueden regalarlo | `TestPauseFreezesTimer` · `TestMultiplePausesAccumulate` · `TestPausingCannotGainTime` · `TestCompleteOnlyWhenTimeIsUp` |
| Las plantas se marchitan en el instante exacto de su vida y no producen después | `TestPlantWiltsExactlyWhenItsLifeEnds` · `TestWitheredPlantStopsProducingAndTheSiloKeepsWhatItMade` |
| El Silo nunca supera su capacidad ni quita monedas | `TestFullSiloLosesProductionUntilEmptied` · `TestSiloNeverTakesAwayCoins` |
| Descansos y premios calculados por marcas de tiempo | `TestRestCountsDownAndEndsByItself` · `TestNoRestWhenHarvestingLongAfterFinishing` |
| Formato de las marcas que envía la API: UTC, RFC 3339, ≤ 9 decimales (cualquier navegador las lee) | `TestEveryTimestampTheAPISendsIsUTCRFC3339` |
| Zonas horarias del jugador (Libro y estadísticas): cambio de horario de verano y fronteras de mes | `TestDaylightSavingDoesNotDuplicateOrSkipADay` · `TestSpringForwardKeepsWeeksAndStreaksIntact` · `TestMonthBoundariesInFarTimeZones` |
| Jugadas aleatorias con el reloj avanzando y retrocediendo | `TestRandomPlayNeverBreaksTheInvariants` |

## Lo que hacen los scripts de navegador

- `p5_03_time.py`: la hora del equipo sube 10 h, baja 3 días y sube 1 año con el Pomodoro en marcha (el contador sigue a ritmo normal y coincide con el servidor ±2 s); recargar con la hora un mes atrás; cerrar el navegador y volver (±1,6 s frente a los datos guardados, que incluye el redondeo hacia arriba del contador); dos dispositivos (mismo tiempo, un segundo Pomodoro rechazado con 409, la pausa se propaga); servidor caído 6 s (el contador local sigue por el reloj monótono y se corrige al volver); página congelada 6 s; y el Pomodoro termina por el reloj del servidor con la hora del equipo cinco días adelantada.
- `p1_04_timer.py`: la cuenta atrás, la pausa y la recuperación tras recargar.
- `p1_07_alerts.py`: el aviso llega a la hora exacta incluso con la pestaña en segundo plano (temporizador en un Worker).

## Cómo se prueba que las pruebas sirven

Mutaciones introducidas a propósito, que las pruebas deben detectar (todas detectadas):

| Defecto introducido | Detectado por |
| :--- | :--- |
| Las marcas de tiempo usan la zona local del servidor | `TestTheServersTimeZoneNeverMatters`, `TestEveryTimestampTheAPISendsIsUTCRFC3339` |
| La liquidación no avanza el marcador `collected_to` (cuenta los mismos tramos una y otra vez) | `TestPollingNeverChangesWhereTheGameEndsUp`, `TestAskingForTheStateManyTimesAtOneInstantChangesNothing`, `TestTheServerClockJumpingForwardAndBackMidPomodoro`, `TestClockGoingBackwardsNeitherProducesNorDoubleCounts` |
