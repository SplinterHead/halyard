import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { darkTheme, lightTheme, toVuetifyTheme } from '../theme'

export default createVuetify({
  display: {
    mobileBreakpoint: 'md',
  },
  theme: {
    defaultTheme: 'dark',
    themes: {
      dark: toVuetifyTheme(darkTheme),
      light: toVuetifyTheme(lightTheme),
    },
  },
  defaults: {
    VCard: {
      elevation: 0,
    },
    VBtn: {
      elevation: 0,
    },
  },
})
