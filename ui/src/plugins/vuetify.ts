import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'

export default createVuetify({
  theme: {
    defaultTheme: 'dark',
    themes: {
      dark: {
        colors: {
          primary: '#8B5CF6', /* Violet */
          secondary: '#3B82F6', /* Blue */
          accent: '#10B981', /* Emerald */
          background: '#050505',
          surface: '#1E293B', /* Slate 800 */
          error: '#EF4444',
          info: '#3B82F6',
          success: '#10B981',
          warning: '#F59E0B',
        },
      },
    },
  },
  defaults: {
    VCard: {
      elevation: 0,
    },
    VBtn: {
      elevation: 0,
    }
  }
})
