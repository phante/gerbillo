# Tassonomia nel Sito principale, collegamento solo tramite due indici JSON

La fonte di verità delle Entità è la directory `content` di Hugo del Sito principale. Le Foto fanno riferimento alle Entità solo tramite Slug. I due siti si parlano esclusivamente con due file JSON: l'Indice delle Entità (dal Sito principale alla gallery) e l'Indice delle Foto (dalla gallery al Sito principale). Il Backend non scrive mai nel repository del Sito principale. L'accoppiamento resta minimo e ogni sito si può costruire e rilasciare da solo.
