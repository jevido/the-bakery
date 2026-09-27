import { mount } from 'svelte'
import '@fontsource/pixelify-sans/400.css'
import '@fontsource/pixelify-sans/600.css'
import './theme.css'
import App from './App.svelte'

mount(App, { target: document.getElementById('app')! })
